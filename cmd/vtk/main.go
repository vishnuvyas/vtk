// Package main provides a CLI tool for formatting data with pretty printing.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/joho/godotenv"
	"github.com/schollz/progressbar/v3"
	"github.com/vishnuvyas/vtk/internal/finder"
	"github.com/vishnuvyas/vtk/internal/format"
	"github.com/vishnuvyas/vtk/internal/stedi"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: vtk <command> [options]\n\nAvailable commands:\n  format    Format input data (supports -f flag)\n  find      Search for pattern in files (respects .gitignore)\n  glob      List files/directories matching regex pattern")
	}

	// load environment variables
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("unable to load any environment settings, please have a .env file in cwd: %v", err)
	}
	command := os.Args[1]

	switch command {
	case "format":
		return runFormat(os.Args[2:])
	case "find":
		return runFind(os.Args[2:])
	case "glob":
		return runGlob(os.Args[2:])
	case "stedi":
		return runStedi(os.Args[2:])
	default:
		return fmt.Errorf("unknown command: %q\n\nAvailable commands:\n  format    Format input data (supports -f flag)\n  find      Search for pattern in files (respects .gitignore)\n  glob      List files/directories matching regex pattern", command)
	}
}

func runStedi(args []string) error {
	stediCmd := flag.NewFlagSet("stedi", flag.ExitOnError)
	key := stediCmd.String("k", os.Getenv("STEDI_API_KEY"), "stedi api key")
	providerName := stediCmd.String("p", "ResolutionCare", "provider name, defaults to (resolution care)")
	npi := stediCmd.String("npi", "1194121681", "rendering provider npi")
	subscriberCsv := stediCmd.String("s", "subscribers.csv", "csv file with subscriber information for eligibility verification")
	outputJsonl := stediCmd.String("o", "eligibility.jsonl", "output jsonl file containing eligibility information")

	if err := stediCmd.Parse(args); err != nil {
		stediCmd.Usage()
		return fmt.Errorf("error parsing arguments for stedi command: %v", err)
	}

	subscribers, err := stedi.LoadSubscriberInfoCSV(*subscriberCsv)
	if err != nil {
		return fmt.Errorf("unable to process input csv due to %v", err)
	}

	stediClient := stedi.NewStediClient(*providerName, *npi, *key)
	ctx := context.Background()
	outputFile, err := os.Create(*outputJsonl)
	if err != nil {
		return fmt.Errorf("unable to open output file: %v", err)
	}
	defer outputFile.Close()
	bar := progressbar.Default(int64(len(subscribers)))

	for _, subscriber := range subscribers {
		resp, err := stediClient.RealtimeEligibility(ctx, subscriber.StediPayerID, subscriber.Subscriber)
		if err != nil {
			return fmt.Errorf("unable to get stedi realtime eligibility: %v", err)
		}
		outputFile.WriteString(resp + "\n")
		bar.Add(1)
	}
	return nil
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
		return format.JSON(data)
	case "sql":
		return format.SQL(data)
	default:
		return fmt.Errorf("unsupported format: %q", *formatType)
	}
}

func runFind(args []string) error {
	// Create a new flag set for the find command
	findCmd := flag.NewFlagSet("find", flag.ExitOnError)
	symbolSearch := findCmd.Bool("s", false, "search for symbols in code files (typescript, tsx, js, jsx, go, python, sql)")

	// Parse flags
	if err := findCmd.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Get remaining arguments (pattern and optional directory)
	remainingArgs := findCmd.Args()
	if len(remainingArgs) < 1 {
		return fmt.Errorf("usage: vtk find [-s] <pattern> [directory]\n\nSearch for a regex pattern in files\n  -s    search for symbols in code files")
	}

	pattern := remainingArgs[0]
	dir := "."

	// Optional directory argument
	if len(remainingArgs) > 1 {
		dir = remainingArgs[1]
	}

	// Perform search (symbol or text)
	var results []finder.Result
	var err error

	if *symbolSearch {
		results, err = finder.FindSymbols(dir, pattern)
	} else {
		results, err = finder.Find(dir, pattern)
	}

	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Format and print results in Emacs compilation mode format
	output := finder.FormatEmacsOutput(results)
	fmt.Print(output)

	return nil
}

func runGlob(args []string) error {
	// Create a new flag set for the glob command
	globCmd := flag.NewFlagSet("glob", flag.ExitOnError)
	matchDirectories := globCmd.Bool("d", false, "match directory names instead of file names")

	// Parse flags
	if err := globCmd.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Get remaining arguments (pattern and optional directory)
	remainingArgs := globCmd.Args()
	if len(remainingArgs) < 1 {
		return fmt.Errorf("usage: vtk glob [-d] <pattern> [directory]\n\nList files/directories matching regex pattern\n  -d    match directory names instead of file names")
	}

	pattern := remainingArgs[0]
	dir := "."

	// Optional directory argument
	if len(remainingArgs) > 1 {
		dir = remainingArgs[1]
	}

	// Perform glob search (files or directories)
	var results []finder.Result
	var err error

	if *matchDirectories {
		results, err = finder.GlobDirectories(dir, pattern)
	} else {
		results, err = finder.GlobFiles(dir, pattern)
	}

	if err != nil {
		return fmt.Errorf("glob failed: %w", err)
	}

	// Print results (one path per line)
	for _, result := range results {
		fmt.Println(result.Path)
	}

	return nil
}
