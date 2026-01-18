package format

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "simple object",
			input:       `{"name":"test","value":123}`,
			expected:    "{\n  \"name\": \"test\",\n  \"value\": 123\n}\n",
			expectError: false,
		},
		{
			name:        "nested object",
			input:       `{"outer":{"inner":"value"},"array":[1,2,3]}`,
			expected:    "{\n  \"array\": [\n    1,\n    2,\n    3\n  ],\n  \"outer\": {\n    \"inner\": \"value\"\n  }\n}\n",
			expectError: false,
		},
		{
			name:        "array",
			input:       `[1,2,3]`,
			expected:    "[\n  1,\n  2,\n  3\n]\n",
			expectError: false,
		},
		{
			name:        "invalid json",
			input:       `{invalid}`,
			expected:    "",
			expectError: true,
		},
		{
			name:        "empty object",
			input:       `{}`,
			expected:    "{}\n",
			expectError: false,
		},
		{
			name:        "empty array",
			input:       `[]`,
			expected:    "[]\n",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := JSON([]byte(tt.input))

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
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

func TestJSON_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errorSubstr string
	}{
		{
			name:        "malformed json",
			input:       `{"key": invalid}`,
			errorSubstr: "failed to parse JSON",
		},
		{
			name:        "incomplete json",
			input:       `{"key":`,
			errorSubstr: "failed to parse JSON",
		},
		{
			name:        "not json",
			input:       `this is not json`,
			errorSubstr: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Suppress stdout for error cases
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			err := JSON([]byte(tt.input))
			if err == nil {
				t.Errorf("expected error containing %q but got no error", tt.errorSubstr)
				return
			}

			if !strings.Contains(err.Error(), tt.errorSubstr) {
				t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
			}
		})
	}
}

func TestJSON_WhitespaceHandling(t *testing.T) {
	// Test that input with various whitespace formats all produce the same output
	inputs := []string{
		`{"a":1,"b":2}`,
		`{ "a" : 1 , "b" : 2 }`,
		`{
			"a": 1,
			"b": 2
		}`,
		`{"a":1,
		"b":2}`,
	}

	expected := "{\n  \"a\": 1,\n  \"b\": 2\n}\n"

	for i, input := range inputs {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := JSON([]byte(input))

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if output != expected {
				t.Errorf("output mismatch for input %d:\nexpected:\n%q\ngot:\n%q", i, expected, output)
			}
		})
	}
}

func TestSQL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "simple SELECT",
			input:       `SELECT id,name FROM users`,
			expected:    "\nSELECT\n  id\n  , name\nFROM users\n",
			expectError: false,
		},
		{
			name:        "SELECT with WHERE",
			input:       `SELECT * FROM users WHERE id=1`,
			expected:    "\nSELECT\n  *\nFROM users\nWHERE id=1\n",
			expectError: false,
		},
		{
			name:        "INSERT statement",
			input:       `INSERT INTO users(name) VALUES('test')`,
			expected:    "\nINSERT INTO users (name)\nVALUES ('test')\n",
			expectError: false,
		},
		{
			name:        "UPDATE statement",
			input:       `UPDATE users SET name='test' WHERE id=1`,
			expected:    "\nUPDATE\n  users\nSET\n  name= 'test'\nWHERE id=1\n",
			expectError: false,
		},
		{
			name:        "DELETE statement",
			input:       `DELETE FROM users WHERE id=1`,
			expected:    "\nDELETE\nFROM users\nWHERE id=1\n",
			expectError: false,
		},
		{
			name:        "empty SQL",
			input:       ``,
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid SQL",
			input:       `INVALID SQL STATEMENT`,
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := SQL([]byte(tt.input))

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check error expectation
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
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

func TestSQL_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errorSubstr string
	}{
		{
			name:        "empty input",
			input:       ``,
			errorSubstr: "failed to parse SQL",
		},
		{
			name:        "whitespace only",
			input:       `   `,
			errorSubstr: "failed to parse SQL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Suppress stdout for error cases
			oldStdout := os.Stdout
			os.Stdout, _ = os.Open(os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			err := SQL([]byte(tt.input))
			if err == nil {
				t.Errorf("expected error containing %q but got no error", tt.errorSubstr)
				return
			}

			if !strings.Contains(err.Error(), tt.errorSubstr) {
				t.Errorf("expected error containing %q, got: %v", tt.errorSubstr, err)
			}
		})
	}
}
