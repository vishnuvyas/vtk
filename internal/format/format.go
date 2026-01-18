// Package format provides formatting functionality for various data formats.
package format

import (
	"encoding/json"
	"fmt"
	"strings"

	sqlfmt "github.com/kanmu/go-sqlfmt"
)

// JSON formats JSON data with pretty printing.
// It takes raw JSON bytes and outputs formatted JSON to stdout.
func JSON(data []byte) error {
	// Parse JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Pretty print JSON
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	// Output to stdout
	fmt.Println(string(prettyJSON))
	return nil
}

// SQL formats SQL statements with proper indentation.
// It takes raw SQL bytes and outputs formatted SQL to stdout.
func SQL(data []byte) error {
	// Trim whitespace
	sql := strings.TrimSpace(string(data))

	// Check for empty input
	if sql == "" {
		return fmt.Errorf("failed to parse SQL: empty input")
	}

	// Format SQL using go-sqlfmt
	formatter := &sqlfmt.Formatter{}
	formatted, err := formatter.Format(sql)
	if err != nil {
		return fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Output to stdout
	fmt.Println(formatted)
	return nil
}
