package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/free5gc/util/mongoapi"
	"go.mongodb.org/mongo-driver/bson"
)

// Same collections as tools/clean/main.go
var subscriberCollections = []string{
	"subscriptionData.authenticationData.authenticationSubscription",
	"subscriptionData.authenticationData.webAuthenticationSubscription",
	"policyData.ues.amData",
	"policyData.ues.smData",
	"subscriptionData.identityData",
	"subscriptionData.provisionedData.amData",
	"subscriptionData.provisionedData.smData",
	"subscriptionData.provisionedData.smfSelectionSubscriptionData",
	"policyData.ues.flowRule",
	"policyData.ues.qosFlow",
	"policyData.ues.chargingData",
}

const backupRoot = "backups"

func main() {
	help := flag.Bool("h", false, "Show help")
	yes := flag.Bool("y", false, "Skip confirmation prompt")
	flag.CommandLine.SetOutput(os.Stdout)
	flag.Parse()

	args := flag.Args()

	if *help || len(args) == 0 {
		fmt.Println("Database Backup/Restore Tool (subscriber data snapshots)")
		fmt.Println("Usage: go run tools/backup/main.go <command> [name] [options]")
		fmt.Println("\nCommands:")
		fmt.Println("  save <name>     Save current subscriber data to backups/<name>/")
		fmt.Println("  load <name>     Wipe current subscriber data and restore from backups/<name>/")
		fmt.Println("  list            List saved backups")
		fmt.Println("  delete <name>   Delete a saved backup")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  go run tools/backup/main.go save backup_1")
		fmt.Println("  go run tools/backup/main.go load backup_1 -y   # restore without prompt")
		fmt.Println("  go run tools/backup/main.go list")
		return
	}

	cmd := args[0]

	switch cmd {
	case "list":
		listBackups()
		return
	case "save", "load", "delete":
		if len(args) < 2 {
			fmt.Println("❌ missing backup name, e.g.: go run tools/backup/main.go " + cmd + " backup_1")
			os.Exit(1)
		}
		name := args[1]
		if err := validateName(name); err != nil {
			fmt.Printf("❌ invalid backup name: %v\n", err)
			os.Exit(1)
		}
		dir := filepath.Join(backupRoot, name)
		switch cmd {
		case "save":
			saveBackup(name, dir)
		case "load":
			if !*yes && !confirm("⚠️  Loading '%s' will DELETE all current subscriber data first. Continue? [y/N] ", name) {
				fmt.Println("Aborted.")
				return
			}
			loadBackup(name, dir)
		case "delete":
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				fmt.Printf("❌ backup '%s' does not exist\n", name)
				os.Exit(1)
			}
			if !*yes && !confirm("⚠️  Delete backup '%s'? [y/N] ", name) {
				fmt.Println("Aborted.")
				return
			}
			if err := os.RemoveAll(dir); err != nil {
				fmt.Printf("❌ failed to delete: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("🗑️  backup '%s' deleted\n", name)
		}
	default:
		fmt.Printf("❌ unknown command '%s' (use save|load|list|delete)\n", cmd)
		os.Exit(1)
	}
}

func validateName(name string) error {
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return fmt.Errorf("must be a simple directory name (no slashes)")
	}
	return nil
}

func confirm(format string, a ...interface{}) bool {
	fmt.Printf(format, a...)
	var answer string
	fmt.Scanln(&answer)
	return answer == "y" || answer == "Y"
}

func connectDB() {
	if err := mongoapi.SetMongoDB("free5gc", "mongodb://localhost:27017"); err != nil {
		fmt.Printf("❌ Failed to connect to MongoDB: %v\n", err)
		os.Exit(1)
	}
}

// fileName maps collection name to a safe filename: dots -> underscores
func fileName(coll string) string {
	return strings.ReplaceAll(coll, ".", "_") + ".json"
}

func saveBackup(name, dir string) {
	connectDB()

	if _, err := os.Stat(dir); err == nil {
		fmt.Printf("⚠️  backup '%s' already exists, overwriting\n", name)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("❌ cannot create %s: %v\n", dir, err)
		os.Exit(1)
	}

	fmt.Printf("💾 Saving backup '%s'...\n\n", name)
	ok, fail := 0, 0
	for _, coll := range subscriberCollections {
		docs, err := mongoapi.RestfulAPIGetMany(coll, bson.M{})
		if err != nil {
			fmt.Printf("  ❌ %-60s Failed: %v\n", coll, err)
			fail++
			continue
		}
		data, err := json.MarshalIndent(docs, "", "  ")
		if err != nil {
			fmt.Printf("  ❌ %-60s Failed: %v\n", coll, err)
			fail++
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, fileName(coll)), data, 0o644); err != nil {
			fmt.Printf("  ❌ %-60s Failed: %v\n", coll, err)
			fail++
			continue
		}
		fmt.Printf("  ✅ %-60s %d documents\n", coll, len(docs))
		ok++
	}

	fmt.Printf("\n📊 Summary: saved %d, failed %d -> %s\n", ok, fail, dir)
}

func loadBackup(name, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fmt.Printf("❌ backup '%s' does not exist (see: go run tools/backup/main.go list)\n", name)
		os.Exit(1)
	}

	connectDB()

	fmt.Printf("📂 Restoring backup '%s'...\n\n", name)
	ok, fail := 0, 0
	for _, coll := range subscriberCollections {
		path := filepath.Join(dir, fileName(coll))
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("  ⚠️  %-60s no backup file, collection left untouched\n", coll)
			continue
		}

		var docs []map[string]interface{}
		if err := json.Unmarshal(data, &docs); err != nil {
			fmt.Printf("  ❌ %-60s Failed to parse %s: %v\n", coll, path, err)
			fail++
			continue
		}

		// wipe current contents, then insert backup docs
		if err := mongoapi.RestfulAPIDeleteMany(coll, nil); err != nil {
			fmt.Printf("  ❌ %-60s Failed to clear: %v\n", coll, err)
			fail++
			continue
		}

		if len(docs) > 0 {
			// drop _id so Mongo generates fresh ObjectIds (avoids string/ObjectID mismatch)
			insert := make([]interface{}, 0, len(docs))
			for _, d := range docs {
				delete(d, "_id")
				insert = append(insert, d)
			}
			if err := mongoapi.RestfulAPIPostMany(coll, nil, insert); err != nil {
				fmt.Printf("  ❌ %-60s Failed to insert: %v\n", coll, err)
				fail++
				continue
			}
		}
		fmt.Printf("  ✅ %-60s %d documents\n", coll, len(docs))
		ok++
	}

	fmt.Printf("\n📊 Summary: restored %d, failed %d (from %s)\n", ok, fail, dir)
}

func listBackups() {
	entries, err := os.ReadDir(backupRoot)
	if err != nil || len(entries) == 0 {
		fmt.Println("No backups found (they are stored in ./backups/)")
		return
	}
	fmt.Printf("📦 Backups in ./%s/:\n\n", backupRoot)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(backupRoot, e.Name())
		files, _ := os.ReadDir(dir)
		total := 0
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, f.Name()))
			if err != nil {
				continue
			}
			var docs []map[string]interface{}
			if json.Unmarshal(data, &docs) == nil {
				total += len(docs)
			}
		}
		fmt.Printf("  • %-20s %2d files, %d documents\n", e.Name(), countJSON(files), total)
	}
}

func countJSON(files []os.DirEntry) int {
	n := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			n++
		}
	}
	return n
}
