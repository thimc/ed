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
			expectedBuffer: []string{"changed", "text"},
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

// TestCmdDeleteLines tests the (d)elete command.
func TestCmdDeleteLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)

	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
	}{
		{
			input:          []byte("1,2d"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"!"},
		},
		{
			input:          []byte("2d"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "!"},
		},
		{
			input:          []byte("d"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world"},
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

// TestCmdChangeLines tests the (i)nsert command.
func TestCmdInsertLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)

	tests := []struct {
		input          []byte
		data           string
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedDot    int
	}{
		{
			input:          []byte("1,3i"),
			data:           "inserted\ntext\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "inserted", "text", "!"},
			expectedDot:    4,
		},
		{
			input:          []byte("1i"),
			data:           "inserted\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"inserted", "hello", "world", "!"},
			expectedDot:    1,
		},
		{
			input:          []byte("i"),
			data:           "inserted\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "inserted", "!"},
			expectedDot:    3,
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

// // TestCmdTransferLines tests the (t)ransfer command.
// func TestCmdTransferLines(t *testing.T) {
// 	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
// 	log.SetOutput(io.Discard)

// 	tests := []struct {
// 		input          []byte
// 		expectError    bool
// 		buffer         []string
// 		expectedBuffer []string
// 		expectedDot    int
// 	}{
// 		{
// 			input:          []byte("1,2t3"),
// 			expectError:    false,
// 			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedBuffer: []string{"hello", "world", "!", "hello", "world", "this", "is", "a", "longer", "file"},
// 			expectedDot:    5,
// 		},
// 		{
// 			input:          []byte("3t4"),
// 			expectError:    false,
// 			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedBuffer: []string{"hello", "world", "!", "this", "!", "is", "a", "longer", "file"},
// 			expectedDot:    5,
// 		},
// 		{
// 			input:          []byte("t5"),
// 			expectError:    false,
// 			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedBuffer: []string{"hello", "world", "!", "this", "is", "file", "a", "longer", "file"},
// 			expectedDot:    6,
// 		},

// 		{
// 			input:          []byte("t"),
// 			expectError:    true,
// 			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedDot:    8,
// 		},
// 		{
// 			input:          []byte("1t"),
// 			expectError:    true,
// 			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
// 			expectedDot:    8,
// 		},
// 	}
// 	for _, test := range tests {
// 		t.Run(string(test.input), func(t *testing.T) {
// 			ted.setupTestFile(test.buffer)
// 			ted.ReadInput(bytes.NewBuffer(test.input))
// 			if err := ted.DoRange(); err != nil && !test.expectError {
// 				t.Fatalf("expected no error, got %s", err)
// 			}
// 			if err := ted.DoCommand(); err != nil && !test.expectError {
// 				t.Fatalf("expected no error, got %s", err)
// 			}
// 			if ted.Dot != test.expectedDot {
// 				t.Fatalf("expected the dot to be %d, got %d", test.expectedDot, ted.Dot)
// 			}
// 			if len(test.expectedBuffer) != len(ted.Lines) {
// 				t.Fatalf("expected the total line count to be %d, got %d",
// 					len(test.expectedBuffer), len(ted.Lines))
// 			}
// 			for i := 0; i < len(ted.Lines); i++ {
// 				if ted.Lines[i] != test.expectedBuffer[i] {
// 					t.Errorf("expected line %d to be '%s', got '%s'",
// 						i, test.expectedBuffer[i], ted.Lines[i])
// 				}
// 			}
// 		})
// 	}
// }
