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

func TestFindSymbols(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test files with various languages
	testFiles := map[string]string{
		"test.go": `package main

func HelloWorld() {
	println("hello")
}

func GoodbyeWorld() {
	println("goodbye")
}

type MyStruct struct {
	name string
}
`,
		"test.ts": `function helloTypescript() {
	console.log("hello");
}

class WorldClass {
	constructor() {}
}

const myVariable = 42;
`,
		"test.py": `def hello_python():
	print("hello")

class WorldPython:
	def __init__(self):
		pass

my_var = 42
`,
		"test.js": `function helloJavaScript() {
	console.log("hello");
}

const worldConst = "world";
`,
		"test.sql": `CREATE TABLE hello_table (
	id INT PRIMARY KEY
);

CREATE FUNCTION world_function()
RETURNS INT AS $$
BEGIN
	RETURN 42;
END;
$$ LANGUAGE plpgsql;
`,
		"README.md": `# Documentation
This file should be ignored in symbol search
`,
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	tests := []struct {
		name            string
		pattern         string
		expectedSymbols []string // symbols that should be found
		unexpectedFiles []string // file types that should NOT be searched
	}{
		{
			name:    "find hello symbols",
			pattern: "(?i)hello", // case-insensitive
			expectedSymbols: []string{
				"HelloWorld",
				"helloTypescript",
				"hello_python",
				"helloJavaScript",
				"hello_table",
			},
			unexpectedFiles: []string{"README.md"},
		},
		{
			name:    "find world symbols",
			pattern: "[Ww]orld",
			expectedSymbols: []string{
				"HelloWorld",
				"GoodbyeWorld",
				"WorldClass",
				"WorldPython",
				"worldConst",
				"world_function",
			},
			unexpectedFiles: []string{"README.md"},
		},
		{
			name:            "no symbol matches",
			pattern:         "nonexistent",
			expectedSymbols: []string{},
			unexpectedFiles: []string{"README.md"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := FindSymbols(tempDir, tt.pattern)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check expected symbols appear
			for _, expectedSymbol := range tt.expectedSymbols {
				found := false
				for _, result := range results {
					if strings.Contains(result.Match, expectedSymbol) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find symbol %q, but didn't", expectedSymbol)
				}
			}

			// Check unexpected files don't appear
			for _, unexpectedFile := range tt.unexpectedFiles {
				for _, result := range results {
					if strings.Contains(result.Path, unexpectedFile) {
						t.Errorf("did not expect matches in %s", unexpectedFile)
					}
				}
			}
		})
	}
}

func TestFindSymbols_UnsupportedFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create files of unsupported types
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("some text"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test.md"), []byte("# markdown"), 0644)

	results, err := FindSymbols(tempDir, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find no symbols since these file types aren't supported
	if len(results) > 0 {
		t.Errorf("expected no results for unsupported file types, got %d", len(results))
	}
}

func TestIsSupportedSymbolFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.go", true},
		{"test.ts", true},
		{"test.tsx", true},
		{"test.js", true},
		{"test.jsx", true},
		{"test.py", true},
		{"test.sql", true},
		{"test.txt", false},
		{"test.md", false},
		{"test.json", false},
		{"test.rs", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := IsSupportedSymbolFile(tt.filename)
			if result != tt.expected {
				t.Errorf("expected IsSupportedSymbolFile(%q) to return %v, got %v",
					tt.filename, tt.expected, result)
			}
		})
	}
}

func TestReplace(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "hello world\nhello there\ngoodbye world",
		"file2.txt":        "no matches here",
		"subdir/file3.go":  "func hello() {\n\tprintln(\"hello\")\n}",
		"ignored/test.txt": "hello ignored",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Create .gitignore
	gitignore := "ignored/\n"
	os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignore), 0644)

	// Test replacement
	results, err := Replace(tempDir, "hello", "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that replacements were made
	if len(results) == 0 {
		t.Error("expected replacements to be made")
	}

	// Verify file1.txt was modified
	content, _ := os.ReadFile(filepath.Join(tempDir, "file1.txt"))
	if !strings.Contains(string(content), "hi world") {
		t.Error("expected 'hello' to be replaced with 'hi' in file1.txt")
	}
	if strings.Contains(string(content), "hello world") {
		t.Error("expected 'hello world' to be completely replaced")
	}

	// Verify file2.txt was not modified
	content, _ = os.ReadFile(filepath.Join(tempDir, "file2.txt"))
	if content == nil || string(content) != "no matches here" {
		t.Error("expected file2.txt to remain unchanged")
	}

	// Verify ignored file was not touched
	content, _ = os.ReadFile(filepath.Join(tempDir, "ignored/test.txt"))
	if !strings.Contains(string(content), "hello ignored") {
		t.Error("expected ignored file to remain unchanged")
	}
}

func TestReplaceSymbol(t *testing.T) {
	tempDir := t.TempDir()

	// Create test Go file with function and calls
	goFile := `package main

func oldName() {
	println("test")
}

func caller() {
	oldName()
	oldName()
}
`
	os.WriteFile(filepath.Join(tempDir, "test.go"), []byte(goFile), 0644)

	// Test semantic replacement
	results, err := ReplaceSymbol(tempDir, "oldName", "newName")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should replace both definition and calls
	if len(results) < 2 {
		t.Errorf("expected at least 2 replacements (definition + calls), got %d", len(results))
	}

	// Verify file was modified
	content, _ := os.ReadFile(filepath.Join(tempDir, "test.go"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "func newName()") {
		t.Error("expected function definition to be renamed")
	}
	if strings.Contains(contentStr, "func oldName()") {
		t.Error("expected old function name to be gone")
	}
	if !strings.Contains(contentStr, "newName()") {
		t.Error("expected function calls to be renamed")
	}
	if strings.Contains(contentStr, "oldName()") {
		t.Error("expected old function calls to be gone")
	}
}

func TestReplaceSymbol_JavaScript(t *testing.T) {
	tempDir := t.TempDir()

	jsFile := `function oldFunc() {
	console.log("test");
}

const x = oldFunc();
oldFunc();
`
	os.WriteFile(filepath.Join(tempDir, "test.js"), []byte(jsFile), 0644)

	results, err := ReplaceSymbol(tempDir, "oldFunc", "newFunc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("expected multiple replacements, got %d", len(results))
	}

	content, _ := os.ReadFile(filepath.Join(tempDir, "test.js"))
	contentStr := string(content)

	if !strings.Contains(contentStr, "function newFunc()") {
		t.Error("expected function definition to be renamed")
	}
	if !strings.Contains(contentStr, "newFunc()") {
		t.Error("expected function calls to be renamed")
	}
}

func TestGlobFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files and directories
	testFiles := map[string]string{
		"file1.go":            "package main",
		"file2.txt":           "text",
		"test_file.go":        "package test",
		"subdir/nested.go":    "package nested",
		"subdir/data.json":    "{}",
		"subdir/deep/test.go": "package deep",
		"ignored/ignore.go":   "package ignored",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	// Create .gitignore
	gitignore := "ignored/\n"
	os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignore), 0644)

	// Test matching .go files
	results, err := GlobFiles(tempDir, `\.go$`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find all .go files except ignored ones
	expectedFiles := []string{
		filepath.Join(tempDir, "file1.go"),
		filepath.Join(tempDir, "test_file.go"),
		filepath.Join(tempDir, "subdir/nested.go"),
		filepath.Join(tempDir, "subdir/deep/test.go"),
	}

	if len(results) != len(expectedFiles) {
		t.Errorf("expected %d files, got %d", len(expectedFiles), len(results))
	}

	// Verify results contain expected files
	resultPaths := make([]string, len(results))
	for i, r := range results {
		resultPaths[i] = r.Path
	}

	for _, expected := range expectedFiles {
		found := false
		for _, path := range resultPaths {
			if path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find %s in results", expected)
		}
	}

	// Verify ignored directory is not included
	for _, r := range results {
		if strings.Contains(r.Path, "ignored") {
			t.Errorf("expected ignored directory to be skipped, but found: %s", r.Path)
		}
	}
}

func TestGlobFiles_InvalidPattern(t *testing.T) {
	tempDir := t.TempDir()

	// Test with invalid regex
	_, err := GlobFiles(tempDir, "[invalid")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}

func TestGlobFiles_NonExistentDirectory(t *testing.T) {
	_, err := GlobFiles("/non/existent/directory", ".*")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestGlobDirectories(t *testing.T) {
	tempDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"pkg/util",
		"pkg/helper",
		"cmd/app",
		"internal/test",
		"test_data",
		"ignored/dir",
	}

	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		// Create a file so directories are not empty
		os.WriteFile(filepath.Join(tempDir, dir, "dummy.txt"), []byte("test"), 0644)
	}

	// Create .gitignore
	gitignore := "ignored/\n"
	os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignore), 0644)

	// Test matching directories with "test" in name
	results, err := GlobDirectories(tempDir, `test`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find directories with "test" in name
	expectedDirs := []string{
		filepath.Join(tempDir, "internal/test"),
		filepath.Join(tempDir, "test_data"),
	}

	if len(results) != len(expectedDirs) {
		t.Errorf("expected %d directories, got %d", len(expectedDirs), len(results))
	}

	// Verify results
	for _, expected := range expectedDirs {
		found := false
		for _, r := range results {
			if r.Path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected to find directory %s", expected)
		}
	}

	// Verify ignored directory is not included
	for _, r := range results {
		if strings.Contains(r.Path, "ignored") {
			t.Errorf("expected ignored directory to be skipped, but found: %s", r.Path)
		}
	}
}

func TestGlobDirectories_InvalidPattern(t *testing.T) {
	tempDir := t.TempDir()

	_, err := GlobDirectories(tempDir, "[invalid")
	if err == nil {
		t.Error("expected error for invalid regex pattern")
	}
}
