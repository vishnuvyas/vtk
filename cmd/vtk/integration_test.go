package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunFormat_Integration tests the complete argument handling flow
func TestRunFormat_Integration(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		stdin       string
		expected    string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "format with default flag from stdin",
			args:        []string{},
			stdin:       `{"key":"value"}`,
			expected:    "{\n  \"key\": \"value\"\n}\n",
			expectError: false,
		},
		{
			name:        "format with explicit -f json flag",
			args:        []string{"-f", "json"},
			stdin:       `{"a":1,"b":2}`,
			expected:    "{\n  \"a\": 1,\n  \"b\": 2\n}\n",
			expectError: false,
		},
		{
			name:        "format with unsupported format",
			args:        []string{"-f", "xml"},
			stdin:       `{"key":"value"}`,
			expected:    "",
			expectError: true,
			errorSubstr: "unsupported format",
		},
		{
			name:        "format with invalid json from stdin",
			args:        []string{},
			stdin:       `{invalid json}`,
			expected:    "",
			expectError: true,
			errorSubstr: "failed to parse JSON",
		},
		{
			name:        "format with empty stdin",
			args:        []string{},
			stdin:       ``,
			expected:    "",
			expectError: true,
			errorSubstr: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore stdin
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			// Create pipe for stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			w.Write([]byte(tt.stdin))
			w.Close()

			// Capture stdout
			oldStdout := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Run the format command
			err := runFormat(tt.args)

			// Restore stdout and read output
			wOut.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, rOut)
			output := buf.String()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check output
			if output != tt.expected {
				t.Errorf("output mismatch:\nexpected:\n%q\ngot:\n%q", tt.expected, output)
			}
		})
	}
}

// TestRunFormat_FileInput tests reading from file arguments
func TestRunFormat_FileInput(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		args        []string
		fileContent string
		expected    string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "format from file",
			args:        []string{filepath.Join(tempDir, "test1.json")},
			fileContent: `{"name":"test","count":42}`,
			expected:    "{\n  \"count\": 42,\n  \"name\": \"test\"\n}\n",
			expectError: false,
		},
		{
			name:        "format from file with -f json",
			args:        []string{"-f", "json", filepath.Join(tempDir, "test2.json")},
			fileContent: `[1,2,3]`,
			expected:    "[\n  1,\n  2,\n  3\n]\n",
			expectError: false,
		},
		{
			name:        "format from non-existent file",
			args:        []string{filepath.Join(tempDir, "nonexistent.json")},
			fileContent: "",
			expected:    "",
			expectError: true,
			errorSubstr: "failed to open file",
		},
		{
			name:        "format from file with invalid json",
			args:        []string{filepath.Join(tempDir, "invalid.json")},
			fileContent: `{this is not valid}`,
			expected:    "",
			expectError: true,
			errorSubstr: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if content is provided
			if tt.fileContent != "" {
				filePath := tt.args[len(tt.args)-1] // Last arg is always the file path
				err := os.WriteFile(filePath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run the format command
			err := runFormat(tt.args)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check output
			if output != tt.expected {
				t.Errorf("output mismatch:\nexpected:\n%q\ngot:\n%q", tt.expected, output)
			}
		})
	}
}

// TestRun_CommandRouting tests the main command routing logic
func TestRun_CommandRouting(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "no command provided",
			args:        []string{"vtk"},
			expectError: true,
			errorSubstr: "usage: vtk <command>",
		},
		{
			name:        "unknown command",
			args:        []string{"vtk", "unknown"},
			expectError: true,
			errorSubstr: "unknown command",
		},
		{
			name:        "invalid command",
			args:        []string{"vtk", "foobar"},
			expectError: true,
			errorSubstr: "unknown command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args and restore after test
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set test args
			os.Args = tt.args

			// Suppress output
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			oldStdin := os.Stdin
			os.Stdin, _ = os.Open(os.DevNull)
			defer func() { os.Stdin = oldStdin }()

			// Run
			err := run()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestRunFormat_FlagParsing tests various flag combinations
func TestRunFormat_FlagParsing(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.json")
	testContent := `{"test":"data"}`
	os.WriteFile(testFile, []byte(testContent), 0644)

	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid flag before file",
			args:        []string{"-f", "json", testFile},
			expectError: false,
		},
		{
			name:        "default format",
			args:        []string{testFile},
			expectError: false,
		},
		{
			name:        "unsupported format type",
			args:        []string{"-f", "yaml", testFile},
			expectError: true,
			errorSubstr: "unsupported format",
		},
		{
			name:        "multiple files not supported",
			args:        []string{testFile, testFile},
			expectError: false, // Second file is ignored, first one is processed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Suppress output
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			// Run the format command
			err := runFormat(tt.args)

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestRunFormat_ComplexJSON tests formatting of complex nested structures
func TestRunFormat_ComplexJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "deeply nested object",
			input: `{"a":{"b":{"c":{"d":"value"}}}}`,
			expected: `{
  "a": {
    "b": {
      "c": {
        "d": "value"
      }
    }
  }
}
`,
		},
		{
			name:  "mixed arrays and objects",
			input: `{"items":[{"id":1,"name":"first"},{"id":2,"name":"second"}],"count":2}`,
			expected: `{
  "count": 2,
  "items": [
    {
      "id": 1,
      "name": "first"
    },
    {
      "id": 2,
      "name": "second"
    }
  ]
}
`,
		},
		{
			name:  "array of arrays",
			input: `[[1,2],[3,4],[5,6]]`,
			expected: `[
  [
    1,
    2
  ],
  [
    3,
    4
  ],
  [
    5,
    6
  ]
]
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore stdin
			oldStdin := os.Stdin
			defer func() { os.Stdin = oldStdin }()

			// Create pipe for stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			w.Write([]byte(tt.input))
			w.Close()

			// Capture stdout
			oldStdout := os.Stdout
			rOut, wOut, _ := os.Pipe()
			os.Stdout = wOut

			// Run the format command
			err := runFormat([]string{})

			// Restore stdout and read output
			wOut.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, rOut)
			output := buf.String()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check output
			if output != tt.expected {
				t.Errorf("output mismatch:\nexpected:\n%q\ngot:\n%q", tt.expected, output)
			}
		})
	}
}
