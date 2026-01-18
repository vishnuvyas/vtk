// Package main provides a CLI tool for formatting data with pretty printing.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	sqlfmt "github.com/kanmu/go-sqlfmt"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: vtk <command> [options]\n\nAvailable commands:\n  format    Format input data (supports -f flag)")
	}

	command := os.Args[1]

	switch command {
	case "format":
		return runFormat(os.Args[2:])
	default:
		return fmt.Errorf("unknown command: %q\n\nAvailable commands:\n  format    Format input data (supports -f flag)", command)
	}
}

func runFormat(args []string) error {
	// Create a new flag set for the format command
	formatCmd := flag.NewFlagSet("format", flag.ExitOnError)
	formatType := formatCmd.String("f", "json", "output format (json, sql)")

	// Parse flags
	if err := formatCmd.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Validate format type
	if *formatType != "json" && *formatType != "sql" {
		return fmt.Errorf("unsupported format: %q (supported: json, sql)", *formatType)
	}

	var input io.Reader

	// Determine input source: file or stdin
	remainingArgs := formatCmd.Args()
	if len(remainingArgs) > 0 {
		// Read from file
		filePath := remainingArgs[0]
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %w", filePath, err)
		}
		defer file.Close()
		input = file
	} else {
		// Read from stdin
		input = os.Stdin
	}

	// Read all input
	data, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Format based on type
	switch *formatType {
	case "json":
		return formatJSON(data)
	case "sql":
		return formatSQL(data)
	default:
		return fmt.Errorf("unsupported format: %q", *formatType)
	}
}

func formatJSON(data []byte) error {
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

func formatSQL(data []byte) error {
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
