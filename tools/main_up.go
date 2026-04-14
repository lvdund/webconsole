package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/free5gc/util/mongoapi"
	"github.com/free5gc/webconsole/backend/logger"
	"github.com/free5gc/webconsole/tools/data"
)

func main() {
	var (
		count     = flag.Int("n", 10, "Number of subscribers to create")
		startIMSI = flag.String("start", "imsi-208930000000001", "Starting IMSI (with imsi- prefix)")
		plmnID    = flag.String("plmn", "20893", "PLMN ID")
		useOP     = flag.Bool("op", false, "Use OP (Operator Key)")
		useOPC    = flag.Bool("opc", false, "Use OPC (Operator Code)")
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

	// Create subscribers
	successCount := 0
	failCount := 0
	currentIMSI := *startIMSI

	for i := 0; i < *count; i++ {
		userNumber := i + 1
		fmt.Printf("Creating subscriber %d/%d (IMSI: %s)...", userNumber, *count, currentIMSI)

		err := data.PostSub(&data.SubsData, currentIMSI, *plmnID)
		if err != nil {
			fmt.Printf(" ❌ Failed: %v\n", err)
			failCount++
		} else {
			fmt.Printf(" ✅ Success\n")
			successCount++
		}

		// Generate next IMSI for next iteration
		if i < *count-1 { // Don't generate next IMSI for the last iteration
			nextIMSI, err := data.NextIMSI(currentIMSI)
			if err != nil {
				log.Fatalf("Failed to generate next IMSI: %v", err)
			}
			currentIMSI = nextIMSI
		}

		// Small delay to avoid overwhelming the database
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  ✅ Successful: %d\n", successCount)
	fmt.Printf("  ❌ Failed: %d\n", failCount)
	fmt.Printf("  📈 Success rate: %.1f%%\n", float64(successCount)/float64(*count)*100)
}
