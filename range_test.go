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
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
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
			var b bytes.Buffer
			ted.ReadInput(bytes.NewBuffer(test.input))
			ted.out = &b
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected the range to be valid but failed: %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected the command to be valid but failed: %s", err)
			}
			if ted.Start != test.expectedStart {
				t.Errorf("expected start position %d, got %d",
					test.expectedStart, ted.Start)
			}
			if ted.End != test.expectedEnd {
				t.Errorf("expected end position %d, got %d", test.expectedEnd, ted.End)
			}
			// if ted.Dot != test.expectedDot {
			// 	t.Errorf("expected dot position %d, got %d", test.expectedDot, ted.Dot)
			// }
			if ted.addrcount != test.expectedAddrCount {
				t.Errorf("expected internal address count %d, got %d",
					test.expectedAddrCount, ted.addrcount)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
		})
	}
}

func (ed *Editor) setupTestFile(buf []string) {
	ed.Lines = buf
	ed.Path = "test"
	ed.Dot = len(buf)
	ed.Start = ed.Dot
	ed.End = ed.Dot
	ed.addr = -1
}

func (ed *Editor) removeDummyFile(fname string) {
	if _, err := os.Stat(fname); err == nil {
		if err := os.Remove(fname); err != nil {
			panic(err)
		}
	}
}

func (ed *Editor) createDummyFile(fname string) {
	ed.removeDummyFile(fname)
	file, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	for _, ln := range dummyFile {
		_, err := file.WriteString(ln+"\n")
		if err != nil {
			panic(err)
		}
	}
}
