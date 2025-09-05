package WebUI

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func LogToFilePrettyJSON(s *SubsData, filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Add timestamp header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	file.WriteString(fmt.Sprintf("\n=== SubsData Log Entry - %s ===\n", timestamp))

	// Pretty print JSON with indentation
	jsonData, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	file.WriteString(string(jsonData))
	file.WriteString("\n" + strings.Repeat("=", 50) + "\n")

	return nil
}
