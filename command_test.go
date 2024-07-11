package main

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestCmdAppendLines tests the append (a) command.
func TestCmdAppendLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		data           string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1,3a",
			data:           "A\nB\nC\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "C", "A", "B", "C"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          "2a",
			data:           "A\nB\nC\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "A", "B", "C", "C"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          "a",
			data:           "D\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "C", "D"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          "2a",
			data:           "C\n.",
			buffer:         []string{"A", "B"},
			expectedBuffer: []string{"A", "B", "C"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.in = strings.NewReader(test.input + "\n" + test.data)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q for %q %q", err, test.input, test.data)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q", i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdBang tests the shell ! command.
func TestCmdBang(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		expectedError  error
		expectedOutput string
	}{
		{
			input:         "!",
			expectedError: ErrNoCmd,
		},
		{
			input:          "!ls *.go | wc -l", // probably a bad idea
			expectedError:  nil,
			expectedOutput: "6\n!\n",
		},
		{
			input:          "!",
			expectedError:  nil,
			expectedOutput: "6\n!\n",
		},
		{
			input:          "! ",
			expectedError:  nil,
			expectedOutput: "6\n!\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.printErrors = true
			ted.err = &b
			ted.readInput(strings.NewReader(test.input))
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != test.expectedError {
				t.Fatalf("expected no error, got %q for cmd %q", err, test.input)
			}
			if strings.TrimLeft(b.String(), " ") != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
		})
	}
}

// TestCmdChangeLines tests the change (c) command.
func TestCmdChangeLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		data           string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1,3c",
			data:           "D\nE\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"D", "E"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          ",c",
			data:           "A\nB\nC\n.\n",
			buffer:         []string{"C"},
			expectedBuffer: []string{"A", "B", "C"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "1c",
			data:           "D\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"D", "B", "C"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "c",
			data:           "D\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "D"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.printErrors = true
			ted.in = strings.NewReader(test.input + "\n" + test.data)
			ted.setupTestFile(test.buffer)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q for cmd %q", err, test.input)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdDeleteLines tests the delete (d) command.
func TestCmdDeleteLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		expectedError  error
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1,2d",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"C"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "2d",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "C"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          "d",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B"},
			expectedStart:  2,
			expectedEnd:    2,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != test.expectedError {
				t.Fatalf("expected error %q, got %q for cmd %q", test.expectedError, err, test.input)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q", i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdEdit tests the edit (e) command.
func TestCmdEdit(t *testing.T) {
	var (
		ted  = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
		path = "dummy"
	)
	ted.createDummyFile(path)
	defer ted.removeDummyFile(path)
	tests := []struct {
		input          string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "e " + path,
			expectedOutput: "52\n",
			expectedStart:  26,
			expectedEnd:    26,
		},
		{
			input:          "e !ls main.go",
			expectedOutput: "8\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.err = &b
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdFile tests the file name (f) command.
func TestCmdFile(t *testing.T) {
	var (
		ted           = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
		b             bytes.Buffer
		expectedError = ErrNoFileName
		path          = "dummy"
	)

	ted.in = strings.NewReader("f")
	ted.printErrors = true
	if err := ted.Do(); err != expectedError && err != nil {
		t.Fatalf("expected error %q, got %q", expectedError, err)
	}

	if err := ted.createDummyFile(path); err != nil {
		t.Fatalf("failed to create a dummy file: %s", err)
	}
	defer ted.removeDummyFile(path)

	ted.in = strings.NewReader("e " + path)
	if err := ted.Do(); err != nil {
		t.Fatalf("expected no error,  got %q", err)
	}
	ted.err = &b
	ted.in = strings.NewReader("f")
	if err := ted.Do(); err != nil {
		t.Fatalf("expected no error,  got %q", err)
	}
	if b.String() != path+"\n" {
		t.Fatalf("expected output to be %q, got %q", path, b.String())
	}
}

// TestCmdGlobal tests the global (g) and inverse global (v) command.
func TestCmdGlobal(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          ",g/A/d",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"B1", "B2", "B3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          ",g/A/",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          ",v/A/",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          ",v/A/d",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "A2", "A3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "2,$g|A|d",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "B1", "B2", "B3"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          "2,$v|A|d",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "A2", "A3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          ",g/A/s/A/B/g",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"B1", "B1", "B2", "B2", "B3", "B3"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          "3g|A|",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "g|B|d",
			buffer:         []string{"A1", "B1", "A2", "B2", "A3", "B3"},
			expectedBuffer: []string{"A1", "A2", "A3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdInsertLines tests the insert (i) command.
func TestCmdInsertLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		data           string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1,3i",
			data:           "D\nE\n.",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "D", "E", "C"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          "1i",
			data:           "D\n.",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"D", "A", "B", "C"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "i",
			data:           "D\n.",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "D", "C"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "2i",
			data:           "B\n.",
			buffer:         []string{"A", "C"},
			expectedBuffer: []string{"A", "B", "C"},
			expectedStart:  2,
			expectedEnd:    2,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.in = strings.NewReader(test.input + "\n" + test.data)
			ted.setupTestFile(test.buffer)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdJoinLines tests the join (j) command.
func TestCmdJoinLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		expectedError  error
		buffer         []string
		expectedBuffer []string
	}{
		{
			input:          "1,2j",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"AB", "C"},
		},
		{
			input:          ",j",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"ABC"},
		},
		{
			input:          "2j",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "BC"},
		},
		{
			input:          "j",
			expectedError:  ErrInvalidAddress,
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "C"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.printErrors = true
			ted.setupTestFile(test.buffer)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != test.expectedError {
				t.Fatalf("expected error %q, got %q for cmd %q", test.expectedError, err, test.input)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
		})
	}
}

// TestCmdMark tests the mark (k) command and the ' address symbol.
func TestCmdMark(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	ted.setupTestFile(dummyFile)
	tests := []struct {
		input         [][]byte
		expectedStart []int
		expectedEnd   []int
	}{
		{
			input:         [][]byte{[]byte("3ka"), []byte("5"), []byte("'a")},
			expectedStart: []int{3, 5, 3},
			expectedEnd:   []int{3, 5, 3},
		},
		{
			input:         [][]byte{[]byte("2,5ka"), []byte("'a")},
			expectedStart: []int{2, 5},
			expectedEnd:   []int{5, 5},
		},
		{
			input:         [][]byte{[]byte("1ka"), []byte("5kb"), []byte("'a,'b")},
			expectedStart: []int{1, 5, 1},
			expectedEnd:   []int{1, 5, 5},
		},
	}
	for _, test := range tests {
		t.Run(string(bytes.Join(test.input, []byte(" "))), func(t *testing.T) {
			for n, cmd := range test.input {
				ted.in = bytes.NewBuffer(cmd)
				if err := ted.Do(); err != nil {
					t.Fatalf("expected no error, got %q", err)
				}
				if test.expectedStart[n] != ted.start {
					t.Fatalf("expected start to be %d, got %d", test.expectedStart[n], ted.start)
				}
				if test.expectedEnd[n] != ted.end {
					t.Fatalf("expected start to be %d, got %d", test.expectedEnd[n], ted.end)
				}
			}
		})
	}
}

// TestCmdMoveLines tests the movement (m) command.
func TestCmdMoveLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		expectedError  error
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "4,6m1",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"A", "D", "E", "F", "B", "C"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          "4,6m0",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"D", "E", "F", "A", "B", "C"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "4m6",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "E", "F", "D", "G", "H"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          "m2",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "H", "B", "C", "D", "E", "F", "G"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          "m",
			expectedError:  ErrInvalidCmdSuffix,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedStart:  8,
			expectedEnd:    8,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.printErrors = true
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != test.expectedError {
				t.Fatalf("expected error %q, got %q for cmd %q", test.expectedError, err, test.input)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdExplainError tests the print last error (h) command.
func TestCmdExplainError(t *testing.T) {
	var (
		ted           = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
		expectedError = ErrNoFileName
	)
	ted.printErrors = true
	ted.in = bytes.NewBufferString("f")
	if err := ted.Do(); err != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, err)
	}
	ted.in = bytes.NewBufferString("h")
	if err := ted.Do(); err != expectedError {
		t.Fatalf("expected error %q, got %q", expectedError, err)
	}
}

// TestCmdPrintTotalLines tests the total lines (=) command.
func TestCmdPrintTotalLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		buffer         []string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1,5=",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          "3=",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\n",
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "=",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.out = &b
			ted.printErrors = true
			ted.in = strings.NewReader(test.input)
			ted.setupTestFile(test.buffer)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdPrintLines tests all the print commands (p, l, n).
func TestCmdPrintLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		buffer         []string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1p",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "1n",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "1\tA\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "1l",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A$\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "1,5p",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A\nB\nC\nD\nE\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          "1,5n",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "1\tA\n2\tB\n3\tC\n4\tD\n5\tE\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          "1,5l",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A$\nB$\nC$\nD$\nE$\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          ",p",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A\nB\nC\nD\nE\nF\nG\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          ",n",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "1\tA\n2\tB\n3\tC\n4\tD\n5\tE\n6\tF\n7\tG\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          ",l",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A$\nB$\nC$\nD$\nE$\nF$\nG$\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          "p",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "G\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
		{
			input:          "n",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\tG\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
		{
			input:          "l",
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "G$\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.out = &b
			ted.in = strings.NewReader(test.input)
			ted.setupTestFile(test.buffer)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdRead tests the read (r) command.
func TestCmdRead(t *testing.T) {
	var (
		ted  = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
		path = "dummy"
		b    bytes.Buffer
	)
	ted.createDummyFile(path)
	defer ted.removeDummyFile(path)
	tests := []struct {
		input              string
		expectedOutput     string
		expectedStart      int
		expectedEnd        int
		expectedTotalLines int
	}{
		{
			input:              "r " + path,
			expectedOutput:     "52\n",
			expectedStart:      52,
			expectedEnd:        52,
			expectedTotalLines: 52,
		},
		{
			input:              "r !ls main.go",
			expectedOutput:     "8\n",
			expectedStart:      27,
			expectedEnd:        27,
			expectedTotalLines: 27,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			b.Reset()
			ted.err = &b
			ted.setupTestFile(dummyFile)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
			if test.expectedTotalLines != len(ted.Lines) {
				t.Fatalf("expected the total lines to be %d, got %d", test.expectedTotalLines, len(ted.Lines))
			}
		})
	}
}

// TestCmdScroll tests the scroll (z) command.
func TestCmdScroll(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdin(nil), WithStdout(&b), WithStderr(io.Discard))
	)
	ted.setupTestFile(dummyFile)
	tests := []struct {
		input          string
		expectedStart  int
		expectedEnd    int
		expectedOutput string
	}{
		{
			input:          "1,4z5",
			expectedStart:  9,
			expectedEnd:    9,
			expectedOutput: "D\nE\nF\nG\nH\nI\n",
		},
		{
			input:          "z",
			expectedStart:  15,
			expectedEnd:    15,
			expectedOutput: "J\nK\nL\nM\nN\nO\n",
		},
		{
			input:          "5z3",
			expectedStart:  8,
			expectedEnd:    8,
			expectedOutput: "E\nF\nG\nH\n",
		},
		{
			input:          "z20",
			expectedStart:  26,
			expectedEnd:    26,
			expectedOutput: "I\nJ\nK\nL\nM\nN\nO\nP\nQ\nR\nS\nT\nU\nV\nW\nX\nY\nZ\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			b.Reset()
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
		})
	}
}

// TestCmdSearch tests the search commands / and ?.
func TestCmdSearch(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdin(nil), WithStdout(&b), WithStderr(io.Discard))
	)
	ted.setupTestFile(dummyFile)
	tests := []struct {
		input          string
		expectedStart  int
		expectedEnd    int
		expectedOutput string
	}{
		{
			input:          "/A/",
			expectedStart:  1,
			expectedEnd:    1,
			expectedOutput: "A\n",
		},
		{
			input:          "/A",
			expectedStart:  1,
			expectedEnd:    1,
			expectedOutput: "A\n",
		},
		{
			input:          "/A/,/F/p",
			expectedStart:  1,
			expectedEnd:    6,
			expectedOutput: "A\nB\nC\nD\nE\nF\n",
		},
		{
			input:          "?D?p",
			expectedStart:  4,
			expectedEnd:    4,
			expectedOutput: "D\n",
		},
		{
			input:          "?C?,.p",
			expectedStart:  3,
			expectedEnd:    4,
			expectedOutput: "C\nD\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			b.Reset()
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
		})
	}
}

// TestCmdSubstitute tests the substitute (s) command.
func TestCmdSubstitute(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		input          string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          ",s/A/B",
			buffer:         []string{"A B", "A A B", "A A A B"},
			expectedBuffer: []string{"B B", "B A B", "B A A B"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          ",s/A/B/g",
			buffer:         []string{"A B", "A A B", "A A A B"},
			expectedBuffer: []string{"B B", "B B B", "B B B B"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          ",s/A/B/2",
			buffer:         []string{"A B", "A A B", "A A A B"},
			expectedBuffer: []string{"A B", "A B B", "A B A B"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "3s/A/B/",
			buffer:         []string{"A B", "A A B", "A A A B"},
			expectedBuffer: []string{"A B", "A A B", "B A A B"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "3s/A/B/g",
			buffer:         []string{"A B", "A A B", "A A A B"},
			expectedBuffer: []string{"A B", "A A B", "B B B B"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          "3s/A/B/1",
			buffer:         []string{"A B", "A A B", "A A A B"},
			expectedBuffer: []string{"A B", "A A B", "B A A B"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdToggleError tests the toggle error (H) command.
func TestCmdToggleError(t *testing.T) {
	// TODO: TestCmdToggleError
}

// TestCmdTogglePrompt tests the toggle prompt (P) command.
func TestCmdTogglePrompt(t *testing.T) {
	// TODO: TestCmdTogglePrompt
}

// TestCmdTransferLines tests the transfer (t) command.
func TestCmdTransferLines(t *testing.T) {
	var ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
	ted.printErrors = true
	tests := []struct {
		input          string
		expectedError  error
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          "1,2t3",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "A", "B", "D", "E", "F", "G", "H"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          "3t4",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "D", "C", "E", "F", "G", "H"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          "t5",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "D", "E", "H", "F", "G", "H"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          "t",
			expectedError:  ErrDestinationExpected,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedStart:  8,
			expectedEnd:    8,
		},
		{
			input:          "1t",
			expectedError:  ErrDestinationExpected,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          "1t0",
			expectedError:  nil,
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G", "H"},
			expectedBuffer: []string{"A", "A", "B", "C", "D", "E", "F", "G", "H"},
			expectedStart:  1,
			expectedEnd:    1,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != test.expectedError {
				t.Fatalf("expected error %q, got %q for cmd %q", test.expectedError, err, test.input)
			}
			if len(test.expectedBuffer) != len(ted.Lines) {
				t.Fatalf("expected the total line count to be %d, got %d",
					len(test.expectedBuffer), len(ted.Lines))
			}
			for i := 0; i < len(ted.Lines); i++ {
				if ted.Lines[i] != test.expectedBuffer[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedBuffer[i], ted.Lines[i])
				}
			}
			if test.expectedStart != ted.start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.start)
			}
			if test.expectedEnd != ted.end {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.end)
			}
		})
	}
}

// TestCmdUndo tests the undo (u) command.
func TestCmdUndo(t *testing.T) {
	var ted *Editor
	tests := []struct {
		input          []string
		data           []string
		expectedBuffer [][]string
		expectedStart  []int
		expectedEnd    []int
	}{
		{
			input:          []string{"a", "a", "u"},
			data:           []string{"A\nB\n.", "C\n.", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"A", "B", "C"}, {"A", "B"}},
			expectedStart:  []int{2, 3, 2},
			expectedEnd:    []int{2, 3, 2},
		},
		{
			input:          []string{"a", "a", "u"},
			data:           []string{"A\nB\n.", "C\nD\nE\n.", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"A", "B", "C", "D", "E"}, {"A", "B"}},
			expectedStart:  []int{2, 5, 2},
			expectedEnd:    []int{2, 5, 2},
		},
		{
			input:          []string{"a", "1,2c", "u"},
			data:           []string{"A\nB\n.", "C\n.", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"C"}, {"A", "B"}},
			expectedStart:  []int{2, 1, 2},
			expectedEnd:    []int{2, 1, 2},
		},
		{
			input:          []string{"a", "1,2c", "u"},
			data:           []string{"A\nB\n.", "C\nD\n.", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"C", "D"}, {"A", "B"}},
			expectedStart:  []int{2, 2, 2},
			expectedEnd:    []int{2, 2, 2},
		},
		{
			input:          []string{"a", "d", "u"},
			data:           []string{"A\nB\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"A"}, {"A", "B"}},
			expectedStart:  []int{2, 1, 2},
			expectedEnd:    []int{2, 1, 2},
		},
		{
			input:          []string{"a", "1,2d", "u"},
			data:           []string{"A\nB\nC\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C"}, {"C"}, {"A", "B", "C"}},
			expectedStart:  []int{3, 1, 3},
			expectedEnd:    []int{3, 1, 3},
		},
		{
			input:          []string{"a", "2i", "u"},
			data:           []string{"A\nC\n.", "B\n.", ""},
			expectedBuffer: [][]string{{"A", "C"}, {"A", "B", "C"}, {"A", "C"}},
			expectedStart:  []int{2, 2, 2},
			expectedEnd:    []int{2, 2, 2},
		},
		{
			input:          []string{"a", "2i", "u"},
			data:           []string{"A\nE\n.", "B\nC\nD\n.", ""},
			expectedBuffer: [][]string{{"A", "E"}, {"A", "B", "C", "D", "E"}, {"A", "E"}},
			expectedStart:  []int{2, 4, 2},
			expectedEnd:    []int{2, 4, 2},
		},
		{
			input:          []string{"a", "1,2j", "u"},
			data:           []string{"A\nB\nC\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C"}, {"AB", "C"}, {"A", "B", "C"}},
			expectedStart:  []int{3, 1, 3},
			expectedEnd:    []int{3, 1, 3},
		},
		{
			input:          []string{"a", "2,4j", "u"},
			data:           []string{"A\nB\nC\nD\nE\nF\nG\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C", "D", "E", "F", "G"}, {"A", "BCD", "E", "F", "G"}, {"A", "B", "C", "D", "E", "F", "G"}},
			expectedStart:  []int{7, 2, 7},
			expectedEnd:    []int{7, 2, 7},
		},
		{
			input:          []string{"a", "2m0", "u"},
			data:           []string{"A\nB\nC\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C"}, {"B", "A", "C"}, {"A", "B", "C"}},
			expectedStart:  []int{3, 1, 3},
			expectedEnd:    []int{3, 1, 3},
		},
		{
			input:          []string{"a", "2,3m0", "u"},
			data:           []string{"A\nB\nC\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C"}, {"B", "C", "A"}, {"A", "B", "C"}},
			expectedStart:  []int{3, 2, 3},
			expectedEnd:    []int{3, 2, 3},
		},
		{
			input:          []string{"a", "r !ls main.go", "u"},
			data:           []string{"A\n.", "", ""},
			expectedBuffer: [][]string{{"A"}, {"A", "main.go"}, {"A"}},
			expectedStart:  []int{1, 2, 1},
			expectedEnd:    []int{1, 2, 1},
		},
		{
			input:          []string{"a", "1r !ls main.go", "u"},
			data:           []string{"A\nB\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"A", "main.go", "B"}, {"A", "B"}},
			expectedStart:  []int{2, 2, 2},
			expectedEnd:    []int{2, 2, 2},
		},
		{
			input:          []string{"a", "1t2", "u"},
			data:           []string{"A\nB\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B"}, {"A", "B", "A"}, {"A", "B"}},
			expectedStart:  []int{2, 3, 2},
			expectedEnd:    []int{2, 3, 2},
		},
		{
			input:          []string{"a", "1,2t0", "u"},
			data:           []string{"A\nB\nC\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C"}, {"A", "B", "A", "B", "C"}, {"A", "B", "C"}},
			expectedStart:  []int{3, 2, 1},
			expectedEnd:    []int{3, 2, 1},
		},
		{
			input:          []string{"a", "1,2t3", "u"},
			data:           []string{"A\nB\nC\n.", "", ""},
			expectedBuffer: [][]string{{"A", "B", "C"}, {"A", "B", "C", "A", "B"}, {"A", "B", "C"}},
			expectedStart:  []int{3, 5, 3},
			expectedEnd:    []int{3, 5, 3},
		},
	}
	for _, test := range tests {
		ted = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
		for i := 0; i < len(test.data); i++ {
			t.Run(strings.Join(test.input, " "), func(t *testing.T) {
				ted.in = strings.NewReader(test.input[i] + "\n" + test.data[i])
				if err := ted.Do(); err != nil {
					t.Fatalf("expected no error, got %q", err)
				}
				if !reflect.DeepEqual(ted.Lines, test.expectedBuffer[i]) {
					t.Fatalf("expected the file buffer to be %q, got %q", test.expectedBuffer[i], ted.Lines)
				}
				if test.expectedStart[i] != ted.start {
					t.Fatalf("expected start to be %d, got %d", test.expectedStart[i], ted.start)
				}
				if test.expectedEnd[i] != ted.end {
					t.Fatalf("expected end to be %d, got %d", test.expectedEnd[i], ted.end)
				}
			})
		}
	}
}

// TestCmdWrite tests the write commands (w, wq, W).
func TestCmdWrite(t *testing.T) {
	var (
		b    bytes.Buffer
		ted  = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(&b))
		path = "dummy"
	)
	if _, err := os.Stat(path); err == nil {
		ted.removeDummyFile(path)
	}
	// TODO: write commands should not change the start and end position.
	tests := []struct {
		input          string
		buffer         []string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
		expectedLines  []string
	}{
		{
			input:          "w " + path,
			expectedOutput: "52\n",
			expectedStart:  26,
			expectedEnd:    26,
			expectedLines:  dummyFile,
		},
		{
			input:          "10W " + path,
			expectedOutput: "2\n",
			expectedStart:  26,
			expectedEnd:    26,
			expectedLines:  []string{"J"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			b.Reset()
			ted.removeDummyFile(path)
			ted.setupTestFile(dummyFile)
			ted.in = strings.NewReader(test.input)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output %q, got %q", test.expectedOutput, b.String())
			}
			// if test.expectedStart != ted.Start {
			// 	t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			// }
			// if test.expectedEnd != ted.End {
			// 	t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			// }
			lines, err := ted.readFile(path, true, true)
			if err != nil {
				t.Fatalf("expected file %q to exist, got error %q\n", path, err)
			}
			for i := 0; i < len(lines); i++ {
				if lines[i] != test.expectedLines[i] {
					t.Errorf("expected line %d to be %q, got %q",
						i, test.expectedLines[i], lines[i])
				}
			}
			ted.removeDummyFile(path)
		})
	}
}
