// Package finder provides file search functionality with gitignore support.
package finder

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

// Result represents a single match in a file.
type Result struct {
	Path   string
	Line   int
	Column int
	Match  string
}

// Find searches for a pattern in all text files under the given directory,
// respecting .gitignore rules.
func Find(dir string, pattern string) ([]Result, error) {
	// Compile regex pattern
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	// Load .gitignore if it exists
	var gi *ignore.GitIgnore
	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		gi, err = ignore.CompileIgnoreFile(gitignorePath)
		if err != nil {
			// If we can't parse gitignore, continue without it
			gi = nil
		}
	}

	var results []Result

	// Walk the directory tree
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be ignored
			if gi != nil {
				relPath, _ := filepath.Rel(dir, path)
				if relPath != "." && gi.MatchesPath(relPath) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Get relative path for gitignore matching
		relPath, _ := filepath.Rel(dir, path)

		// Check if file is ignored
		if gi != nil && gi.MatchesPath(relPath) {
			return nil
		}

		// Skip binary files
		if IsBinaryFile(path) {
			return nil
		}

		// Search in file
		matches, err := searchFile(path, re)
		if err != nil {
			return nil // Skip files we can't read
		}

		results = append(results, matches...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// searchFile searches for pattern matches in a file.
func searchFile(path string, re *regexp.Regexp) ([]Result, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []Result
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		line := scanner.Text()
		if re.MatchString(line) {
			// Find column position
			loc := re.FindStringIndex(line)
			column := 0
			if len(loc) > 0 {
				column = loc[0]
			}

			results = append(results, Result{
				Path:   path,
				Line:   lineNum,
				Column: column,
				Match:  line,
			})
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// IsBinaryFile checks if a file is binary by looking for null bytes.
func IsBinaryFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read first 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	// Check for null bytes
	return bytes.Contains(buf[:n], []byte{0})
}

// FormatEmacsOutput formats results in Emacs compilation mode format.
// Format: filename:line:column: matching_line
func FormatEmacsOutput(results []Result) string {
	var output strings.Builder

	for _, result := range results {
		// Format: path:line:column: match
		fmt.Fprintf(&output, "%s:%d:%d: %s\n",
			result.Path,
			result.Line,
			result.Column,
			result.Match,
		)
	}

	return output.String()
}
