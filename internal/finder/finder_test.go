package finder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFind(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt":           "hello world\nfoo bar\ntest line",
		"file2.txt":           "no match here\njust some text",
		"subdir/file3.txt":    "hello from subdir\nworld peace",
		"subdir/file4.go":     "package main\nfunc hello() {}\nworld",
		"ignored/secret.txt":  "this should be ignored",
		"binary.bin":          "binary\x00content\x00here",
		".hidden/file.txt":    "hidden hello world",
		"subdir/deep/test.md": "deep hello world\nmarkdown content",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Create .gitignore
	gitignore := "ignored/\n*.bin\n"
	os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignore), 0644)

	tests := []struct {
		name            string
		pattern         string
		expectedFiles   []string // files that should appear in results
		expectedCount   int      // minimum number of matches
		unexpectedFiles []string // files that should NOT appear
	}{
		{
			name:            "simple word search",
			pattern:         "hello",
			expectedFiles:   []string{"file1.txt", "file3.txt", "file4.go", "file.txt", "test.md"},
			expectedCount:   5,
			unexpectedFiles: []string{"secret.txt", "binary.bin"},
		},
		{
			name:            "regex pattern",
			pattern:         "w[oO]rld",
			expectedFiles:   []string{"file1.txt", "file3.txt", "file4.go", "file.txt", "test.md"},
			expectedCount:   5,
			unexpectedFiles: []string{"secret.txt"},
		},
		{
			name:            "no matches",
			pattern:         "nonexistentpattern",
			expectedFiles:   []string{},
			expectedCount:   0,
			unexpectedFiles: []string{},
		},
		{
			name:            "case sensitive",
			pattern:         "Hello",
			expectedFiles:   []string{},
			expectedCount:   0,
			unexpectedFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Find(tempDir, tt.pattern)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check expected files appear
			for _, expectedFile := range tt.expectedFiles {
				found := false
				for _, result := range results {
					if strings.Contains(result.Path, expectedFile) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find matches in %s, but didn't", expectedFile)
				}
			}

			// Check unexpected files don't appear
			for _, unexpectedFile := range tt.unexpectedFiles {
				for _, result := range results {
					if strings.Contains(result.Path, unexpectedFile) {
						t.Errorf("did not expect matches in %s, but found some", unexpectedFile)
					}
				}
			}

			// Check minimum count
			if len(results) < tt.expectedCount {
				t.Errorf("expected at least %d results, got %d", tt.expectedCount, len(results))
			}
		})
	}
}

func TestFind_InvalidPattern(t *testing.T) {
	tempDir := t.TempDir()

	_, err := Find(tempDir, "[invalid")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestFind_NonExistentDirectory(t *testing.T) {
	_, err := Find("/nonexistent/directory/path", "test")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestFormatEmacsOutput(t *testing.T) {
	results := []Result{
		{Path: "/home/user/file.txt", Line: 10, Column: 5, Match: "hello world"},
		{Path: "/home/user/test.go", Line: 42, Column: 1, Match: "func main() {"},
		{Path: "relative/path.txt", Line: 1, Column: 0, Match: "first line"},
	}

	output := FormatEmacsOutput(results)

	// Check format: filename:line:column: match
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 output lines, got %d", len(lines))
	}

	// Verify first line format
	if !strings.Contains(lines[0], "file.txt:10:5:") {
		t.Errorf("expected Emacs format 'file.txt:10:5:', got: %s", lines[0])
	}
	if !strings.Contains(lines[0], "hello world") {
		t.Errorf("expected match text in output, got: %s", lines[0])
	}

	// Verify second line
	if !strings.Contains(lines[1], "test.go:42:1:") {
		t.Errorf("expected Emacs format 'test.go:42:1:', got: %s", lines[1])
	}
}

func TestIsBinaryFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "text file",
			content:  []byte("hello world\nplain text"),
			expected: false,
		},
		{
			name:     "binary with null bytes",
			content:  []byte("binary\x00content\x00here"),
			expected: true,
		},
		{
			name:     "utf8 text",
			content:  []byte("hello 世界\némoji: 🎉"),
			expected: false,
		},
		{
			name:     "empty file",
			content:  []byte(""),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.name)
			os.WriteFile(path, tt.content, 0644)

			result := IsBinaryFile(path)
			if result != tt.expected {
				t.Errorf("expected IsBinaryFile to return %v, got %v", tt.expected, result)
			}
		})
	}
}
