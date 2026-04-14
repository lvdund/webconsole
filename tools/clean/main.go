package main

import (
	"flag"
	"fmt"

	"github.com/free5gc/util/mongoapi"
	"github.com/free5gc/webconsole/backend/logger"
)

func main() {
	var (
		help    = flag.Bool("h", false, "Show help")
		confirm = flag.Bool("y", false, "Skip confirmation prompt")
	)
	flag.Parse()

	if *help {
		fmt.Println("Database Cleanup Tool (Direct MongoDB)")
		fmt.Println("Usage: go run tools/clean/main.go [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExample:")
		fmt.Println("  go run tools/clean/main.go          # With confirmation prompt")
		fmt.Println("  go run tools/clean/main.go -y       # Skip confirmation")
		fmt.Println("\nNote: This deletes ALL subscriber data from every collection.")
		return
	}

	if !*confirm {
		fmt.Print("⚠️  This will delete ALL subscriber data. Are you sure? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Aborted.")
			return
		}
	}

	fmt.Println("Starting database cleanup...")

	// Connect to MongoDB
	if err := mongoapi.SetMongoDB("free5gc", "mongodb://localhost:27017"); err != nil {
		logger.InitLog.Errorf("Failed to connect to MongoDB: %+v", err)
		return
	}

	// Collections that use ueId-only filter (no servingPlmnId)
	ueIdOnlyCollections := []struct {
		name string
		coll string
	}{
		{"Authentication Subscription", "subscriptionData.authenticationData.authenticationSubscription"},
		{"Web Authentication Subscription", "subscriptionData.authenticationData.webAuthenticationSubscription"},
		{"AM Policy Data", "policyData.ues.amData"},
		{"SM Policy Data", "policyData.ues.smData"},
		{"Identity Data", "subscriptionData.identityData"},
	}

	// Collections that use ueId + servingPlmnId filter
	ueIdPlmnCollections := []struct {
		name string
		coll string
	}{
		{"AM Data", "subscriptionData.provisionedData.amData"},
		{"SM Data", "subscriptionData.provisionedData.smData"},
		{"SMF Selection Data", "subscriptionData.provisionedData.smfSelectionSubscriptionData"},
		{"Flow Rule Data", "policyData.ues.flowRule"},
		{"QoS Flow Data", "policyData.ues.qosFlow"},
		{"Charging Data", "policyData.ues.chargingData"},
	}

	successCount := 0
	failCount := 0

	for _, c := range ueIdOnlyCollections {
		if err := mongoapi.RestfulAPIDeleteMany(c.coll, nil); err != nil {
			fmt.Printf("  ❌ %-35s Failed: %v\n", c.name, err)
			failCount++
		} else {
			fmt.Printf("  ✅ %-35s Cleared\n", c.name)
			successCount++
		}
	}

	for _, c := range ueIdPlmnCollections {
		if err := mongoapi.RestfulAPIDeleteMany(c.coll, nil); err != nil {
			fmt.Printf("  ❌ %-35s Failed: %v\n", c.name, err)
			failCount++
		} else {
			fmt.Printf("  ✅ %-35s Cleared\n", c.name)
			successCount++
		}
	}

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("  ✅ Collections cleared: %d\n", successCount)
	fmt.Printf("  ❌ Failed: %d\n", failCount)

}
