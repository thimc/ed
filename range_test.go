package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

var dummyFile = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
}

// TestRange() runs tests on the range parser and verifies the start, end and
// dot position. It also compares the output.
func TestRange(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdin(nil), WithStdout(&b), WithStderr(io.Discard))
	)
	ted.setupTestFile(dummyFile)
	tests := []struct {
		input             []byte
		expectedStart     int
		expectedEnd       int
		expectedDot       int
		expectedAddrCount int
		expectedOutput    string
	}{
		{
			input:             []byte("8"),
			expectedStart:     8,
			expectedEnd:       8,
			expectedAddrCount: 1,
			expectedOutput:    "H\n",
		},
		{
			input:             []byte("1,5"),
			expectedStart:     1,
			expectedEnd:       5,
			expectedAddrCount: 2,
			expectedOutput:    "E\n",
		},
		{
			input:             []byte("+"),
			expectedStart:     6,
			expectedEnd:       6,
			expectedAddrCount: 1,
			expectedOutput:    "F\n",
		},
		{
			input:             []byte("-"),
			expectedStart:     5,
			expectedEnd:       5,
			expectedAddrCount: 1,
			expectedOutput:    "E\n",
		},
		{
			input:             []byte("^"),
			expectedStart:     4,
			expectedEnd:       4,
			expectedAddrCount: 1,
			expectedOutput:    "D\n",
		},
		{
			input:             []byte(".,+5"),
			expectedStart:     4,
			expectedEnd:       9,
			expectedAddrCount: 2,
			expectedOutput:    "I\n",
		},
		{
			input:             []byte("-2,+5"),
			expectedStart:     7,
			expectedEnd:       14,
			expectedAddrCount: 2,
			expectedOutput:    "N\n",
		},
		{
			input:             []byte(","),
			expectedStart:     1,
			expectedEnd:       26,
			expectedAddrCount: 2,
			expectedOutput:    "Z\n",
		},
		{
			input:             []byte("6,%"),
			expectedStart:     6,
			expectedEnd:       26,
			expectedAddrCount: 2,
			expectedOutput:    "Z\n",
		},
		{
			input:             []byte("3;"),
			expectedStart:     3,
			expectedEnd:       26,
			expectedAddrCount: 2,
			expectedOutput:    "Z\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			b.Reset()
			ted.in = bytes.NewBuffer(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if ted.start != test.expectedStart {
				t.Errorf("expected start position %d, got %d",
					test.expectedStart, ted.start)
			}
			if ted.end != test.expectedEnd {
				t.Errorf("expected end position %d, got %d", test.expectedEnd, ted.end)
			}
			// if ted.Dot != test.expectedDot {
			// 	t.Errorf("expected dot position %d, got %d", test.expectedDot, ted.Dot)
			// }
			if ted.addrCount != test.expectedAddrCount {
				t.Errorf("expected internal address count %d, got %d",
					test.expectedAddrCount, ted.addrCount)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
		})
	}
}

// helper function that sets up a buffer in memory
func (ed *Editor) setupTestFile(buf []string) {
	ed.Lines = buf
	ed.path = "test"
	ed.dot = len(buf)
	ed.start = ed.dot
	ed.end = ed.dot
	ed.addr = -1
}

func (ed *Editor) removeDummyFile(fname string) error {
	if err := os.Remove(fname); err != nil {
		return err
	}
	return nil
}

func (ed *Editor) createDummyFile(fname string) error {
	ed.removeDummyFile(fname)
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, ln := range dummyFile {
		_, err := file.WriteString(ln + "\n")
		if err != nil {
			return err
		}
	}
	return nil
}
