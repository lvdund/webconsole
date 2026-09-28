package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/free5gc/util/mongoapi"
	"github.com/free5gc/webconsole/backend/WebUI"
	"github.com/free5gc/webconsole/backend/logger"
	"github.com/free5gc/webconsole/tools/data"
)

func main() {
	defaultWorkers := runtime.NumCPU() * 2
	if defaultWorkers > 32 {
		defaultWorkers = 32
	}

	var (
		count     = flag.Int("n", 10, "Number of subscribers to create")
		startIMSI = flag.String("start", "imsi-208930000000001", "Starting IMSI (with imsi- prefix)")
		plmnID    = flag.String("plmn", "20893", "PLMN ID")
		useOP     = flag.Bool("op", false, "Use OP (Operator Key)")
		useOPC    = flag.Bool("opc", false, "Use OPC (Operator Code)")
		workers   = flag.Int("workers", defaultWorkers, "Number of concurrent MongoDB writers")
		delay     = flag.Duration("delay", 0, "Optional delay after each subscriber write, e.g. 50ms")
		help      = flag.Bool("h", false, "Show help")
	)
	flag.Parse()

	if *help {
		fmt.Println("Bulk Subscriber Upload Tool (Direct MongoDB)")
		fmt.Println("Usage: go run main_up.go [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExample:")
		fmt.Println("  go run main_up.go --op -n 100")
		fmt.Println("  go run main_up.go --opc -n 50 -start imsi-208930000001000 -plmn 20893")
		fmt.Println("\nNote: This tool directly inserts data into MongoDB database")
		return
	}

	if *count <= 0 {
		fmt.Println("Error: -n must be greater than 0")
		return
	}
	if *workers <= 0 {
		*workers = 1
	}
	if *workers > *count {
		*workers = *count
	}

	if !*useOP && !*useOPC {
		*useOP = true
	}

	if *useOP && *useOPC {
		fmt.Println("Error: --op and --opc are mutually exclusive")
		return
	}

	hexKey := "8e27b6af0e692e750f32667a3b14605d"
	if *useOPC {
		data.SubsData.WebAuthenticationSubscription.Milenage.Op.OpValue = ""
		data.SubsData.WebAuthenticationSubscription.Opc.OpcValue = hexKey
	}

	authMode := "OP"
	if *useOPC {
		authMode = "OPC"
	}

	fmt.Printf("Starting bulk subscriber upload (Direct MongoDB)...\n")
	fmt.Printf("Auth mode: %s\n", authMode)
	fmt.Printf("Count: %d subscribers\n", *count)
	fmt.Printf("Starting IMSI: %s\n", *startIMSI)
	fmt.Printf("PLMN ID: %s\n", *plmnID)
	fmt.Printf("Workers: %d\n", *workers)
	if *delay > 0 {
		fmt.Printf("Delay: %s\n", *delay)
	}

	// Connect to MongoDB
	if err := mongoapi.SetMongoDB("free5gc", "mongodb://localhost:27017"); err != nil {
		logger.InitLog.Errorf("Server start err: %+v", err)
		return
	}

	// Get admin tenant ID
	err := data.InitializeAdminTenant()
	if err != nil {
		log.Fatalf("Failed to initialize admin tenant: %v", err)
	}
	fmt.Printf("✅ Admin tenant initialized\n")

	// Pre-generate IMSIs so workers can write concurrently.
	imsis := make([]string, *count)
	currentIMSI := *startIMSI
	for i := 0; i < *count; i++ {
		imsis[i] = currentIMSI
		if i < *count-1 {
			nextIMSI, err := data.NextIMSI(currentIMSI)
			if err != nil {
				log.Fatalf("Failed to generate next IMSI: %v", err)
			}
			currentIMSI = nextIMSI
		}
	}

	// Marshal the template once. Each worker unmarshals into a private copy because
	// PostSub mutates SubsData fields/maps while building MongoDB documents.
	templateJSON, err := json.Marshal(data.SubsData)
	if err != nil {
		log.Fatalf("Failed to prepare subscriber template: %v", err)
	}

	fmt.Printf("Creating subscribers concurrently...\n")
	startTime := time.Now()

	jobs := make(chan string, *workers*2)
	var wg sync.WaitGroup
	var successCount int64
	var failCount int64
	var completed int64
	progressEvery := *count / 20
	if progressEvery < 1 {
		progressEvery = 1
	}
	var printMu sync.Mutex

	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for imsi := range jobs {
				var subData WebUI.SubsData
				if err := json.Unmarshal(templateJSON, &subData); err != nil {
					atomic.AddInt64(&failCount, 1)
					printMu.Lock()
					fmt.Printf("❌ Failed to clone template for %s: %v\n", imsi, err)
					printMu.Unlock()
				} else if err := data.PostSub(&subData, imsi, *plmnID); err != nil {
					atomic.AddInt64(&failCount, 1)
					printMu.Lock()
					fmt.Printf("❌ Failed %s: %v\n", imsi, err)
					printMu.Unlock()
				} else {
					atomic.AddInt64(&successCount, 1)
				}

				done := atomic.AddInt64(&completed, 1)
				if int(done)%progressEvery == 0 || int(done) == *count {
					printMu.Lock()
					fmt.Printf("Progress: %d/%d (✅ %d, ❌ %d)\n", done, *count, atomic.LoadInt64(&successCount), atomic.LoadInt64(&failCount))
					printMu.Unlock()
				}
				if *delay > 0 {
					time.Sleep(*delay)
				}
			}
		}()
	}

	for _, imsi := range imsis {
		jobs <- imsi
	}
	close(jobs)
	wg.Wait()

	elapsed := time.Since(startTime)
	success := atomic.LoadInt64(&successCount)
	failed := atomic.LoadInt64(&failCount)
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  ✅ Successful: %d\n", success)
	fmt.Printf("  ❌ Failed: %d\n", failed)
	fmt.Printf("  📈 Success rate: %.1f%%\n", float64(success)/float64(*count)*100)
	fmt.Printf("  ⏱️  Elapsed: %s (%.1f subscribers/sec)\n", elapsed.Round(time.Millisecond), float64(*count)/elapsed.Seconds())
}
