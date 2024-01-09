package main

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

// TestCmdAppendLines tests the (a)ppend command.
func TestCmdAppendLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)

	tests := []struct {
		input          []byte
		data           string
		expectError    bool
		buffer         []string
		expectedBuffer []string
	}{
		{
			input:          []byte("1,3a"),
			data:           "appended\ntext\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!", "appended", "text"},
		},
		{
			input:          []byte("1a"),
			data:           "appended\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "appended", "world", "!"},
		},
		{
			input:          []byte("a"),
			data:           "appended\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!", "appended"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.in = strings.NewReader(test.data)
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be '%s', got '%s'",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
		})
	}
}


// TestCmdChangeLines tests the (c)hange command.
func TestCmdChangeLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)

	tests := []struct {
		input          []byte
		data           string
		expectError    bool
		buffer         []string
		expectedBuffer []string
	}{
		{
			input:          []byte("1,3c"),
			data:           "changed\ntext\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"changed","text"},
		},
		{
			input:          []byte("1c"),
			data:           "changed\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"changed", "world", "!"},
		},
		{
			input:          []byte("c"),
			data:           "changed\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "changed"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.in = strings.NewReader(test.data)
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be '%s', got '%s'",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
		})
	}

}

// TestCmdJoinLines tests the (j)oin command.
func TestCmdJoinLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)

	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
	}{
		{
			input:          []byte("1,2j"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"helloworld", "!"},
		},
		{
			input:          []byte("2j"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world!"},
		},
		{
			input:          []byte("j"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be '%s', got '%s'",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
		})
	}
}
