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

// Symbol-related functionality

// IsSupportedSymbolFile checks if a file is a supported type for symbol search.
func IsSupportedSymbolFile(filename string) bool {
	ext := filepath.Ext(filename)
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".sql":
		return true
	default:
		return false
	}
}

// FindSymbols searches for symbols matching a pattern in code files.
func FindSymbols(dir string, pattern string) ([]Result, error) {
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
			gi = nil
		}
	}

	var results []Result

	// Walk the directory tree
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories
		if info.IsDir() {
			if gi != nil {
				relPath, _ := filepath.Rel(dir, path)
				if relPath != "." && gi.MatchesPath(relPath) {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if file is supported for symbol search
		if !IsSupportedSymbolFile(path) {
			return nil
		}

		// Get relative path for gitignore matching
		relPath, _ := filepath.Rel(dir, path)
		if gi != nil && gi.MatchesPath(relPath) {
			return nil
		}

		// Extract and search symbols
		symbols, err := extractSymbols(path)
		if err != nil {
			return nil // Skip files we can't parse
		}

		for _, symbol := range symbols {
			if re.MatchString(symbol.Name) {
				results = append(results, Result{
					Path:   path,
					Line:   symbol.Line,
					Column: symbol.Column,
					Match:  symbol.Name,
				})
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// Symbol represents a code symbol (function, class, variable, etc.)
type Symbol struct {
	Name   string
	Line   int
	Column int
	Kind   string // "function", "class", "variable", etc.
}

// extractSymbols extracts symbols from a file based on its language.
func extractSymbols(path string) ([]Symbol, error) {
	ext := filepath.Ext(path)

	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Use simple regex-based extraction for now
	// This is a fallback approach that works reasonably well
	switch ext {
	case ".go":
		return extractGoSymbols(content)
	case ".ts", ".tsx", ".js", ".jsx":
		return extractJSSymbols(content)
	case ".py":
		return extractPythonSymbols(content)
	case ".sql":
		return extractSQLSymbols(content)
	default:
		return nil, nil
	}
}

// extractGoSymbols extracts symbols from Go code using regex
func extractGoSymbols(content []byte) ([]Symbol, error) {
	var symbols []Symbol
	lines := bytes.Split(content, []byte("\n"))

	// Match function definitions: func FuncName( or func (receiver) FuncName(
	funcRe := regexp.MustCompile(`^\s*func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	// Match type definitions: type TypeName struct/interface
	typeRe := regexp.MustCompile(`^\s*type\s+(\w+)\s+(?:struct|interface)`)
	// Match const/var declarations: const/var Name
	varRe := regexp.MustCompile(`^\s*(?:const|var)\s+(\w+)`)

	for i, line := range lines {
		if matches := funcRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "function",
			})
		}
		if matches := typeRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "type",
			})
		}
		if matches := varRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "variable",
			})
		}
	}

	return symbols, nil
}

// extractJSSymbols extracts symbols from JavaScript/TypeScript code
func extractJSSymbols(content []byte) ([]Symbol, error) {
	var symbols []Symbol
	lines := bytes.Split(content, []byte("\n"))

	// Match function definitions
	funcRe := regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`)
	// Match class definitions
	classRe := regexp.MustCompile(`^\s*(?:export\s+)?class\s+(\w+)`)
	// Match const/let/var declarations
	varRe := regexp.MustCompile(`^\s*(?:export\s+)?(?:const|let|var)\s+(\w+)`)
	// Match method definitions
	methodRe := regexp.MustCompile(`^\s*(\w+)\s*\([^)]*\)\s*{`)

	for i, line := range lines {
		if matches := funcRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "function",
			})
		}
		if matches := classRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "class",
			})
		}
		if matches := varRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "variable",
			})
		}
		if matches := methodRe.FindSubmatch(line); matches != nil {
			// Skip if it looks like a function keyword
			if !bytes.Contains(line, []byte("function")) {
				symbols = append(symbols, Symbol{
					Name:   string(matches[1]),
					Line:   i + 1,
					Column: bytes.Index(line, matches[1]),
					Kind:   "method",
				})
			}
		}
	}

	return symbols, nil
}

// extractPythonSymbols extracts symbols from Python code
func extractPythonSymbols(content []byte) ([]Symbol, error) {
	var symbols []Symbol
	lines := bytes.Split(content, []byte("\n"))

	// Match function/method definitions: def func_name(
	funcRe := regexp.MustCompile(`^\s*def\s+(\w+)\s*\(`)
	// Match class definitions: class ClassName
	classRe := regexp.MustCompile(`^\s*class\s+(\w+)`)
	// Match variable assignments at module level (simple heuristic)
	varRe := regexp.MustCompile(`^(\w+)\s*=`)

	for i, line := range lines {
		if matches := funcRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "function",
			})
		}
		if matches := classRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "class",
			})
		}
		if matches := varRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: 0,
				Kind:   "variable",
			})
		}
	}

	return symbols, nil
}

// extractSQLSymbols extracts symbols from SQL code
func extractSQLSymbols(content []byte) ([]Symbol, error) {
	var symbols []Symbol
	lines := bytes.Split(content, []byte("\n"))

	// Match CREATE TABLE
	tableRe := regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+(\w+)`)
	// Match CREATE FUNCTION/PROCEDURE
	funcRe := regexp.MustCompile(`(?i)^\s*CREATE\s+(?:FUNCTION|PROCEDURE)\s+(\w+)`)
	// Match CREATE VIEW
	viewRe := regexp.MustCompile(`(?i)^\s*CREATE\s+VIEW\s+(\w+)`)

	for i, line := range lines {
		if matches := tableRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "table",
			})
		}
		if matches := funcRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "function",
			})
		}
		if matches := viewRe.FindSubmatch(line); matches != nil {
			symbols = append(symbols, Symbol{
				Name:   string(matches[1]),
				Line:   i + 1,
				Column: bytes.Index(line, matches[1]),
				Kind:   "view",
			})
		}
	}

	return symbols, nil
}
