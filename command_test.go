package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// TestCmdAppendLines tests the append (a) command.
func TestCmdAppendLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		data           string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,3a"),
			data:           "A\nB\nC\n.\n",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!", "A", "B", "C"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte("2a"),
			data:           "A\nB\nC\n.\n",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "A", "B", "C", "!"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("a"),
			data:           "appended\n.\n",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!", "appended"},
			expectedStart:  4,
			expectedEnd:    4,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.in = strings.NewReader(test.data)
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdBang tests the shell ! command.
func TestCmdBang(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectedError  error
		expectedOutput string
	}{
		{
			input:         []byte("!"),
			expectedError: ErrNoCmd,
		},
		{
			input:          []byte("!ls *.go | wc -l"), // probably a bad idea
			expectedError:  nil,
			expectedOutput: "       6\n!\n",
		},
		{
			input:          []byte("!"),
			expectedError:  nil,
			expectedOutput: "       6\n!\n",
		},
		{
			input:          []byte("! "),
			expectedError:  nil,
			expectedOutput: "       6\n!\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.err = &b
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != test.expectedError {
				t.Fatalf("expected error '%s', got %s", test.expectedError, err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
		})
	}
}

// TestCmdChangeLines tests the change (c) command.
func TestCmdChangeLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		data           string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,3c"),
			data:           "changed\ntext\n.\n",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"changed", "text"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          []byte("1c"),
			data:           "changed\n.\n",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"changed", "world", "!"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("c"),
			data:           "changed\n.\n",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "changed"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.in = strings.NewReader(test.data)
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdDeleteLines tests the delete (d) command.
func TestCmdDeleteLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectedError  error
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,2d"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"!"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("2d"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "!"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          []byte("d"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world"},
			expectedStart:  2,
			expectedEnd:    2,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != test.expectedError {
				t.Fatalf("expected error '%s', got %s", test.expectedError, err)
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdEdit tests the edit (e) command.
func TestCmdEdit(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	var path string = "dummy"
	ted.createDummyFile(path)
	defer ted.removeDummyFile(path)
	tests := []struct {
		input          []byte
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("e " + path),
			expectedOutput: "52\n",
			expectedStart:  26,
			expectedEnd:    26,
		},
		{
			input:          []byte("e !ls main.go"),
			expectedOutput: "8\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.err = &b
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdFile tests the file name (f) command.
func TestCmdFile(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	var b bytes.Buffer
	ted.ReadInput(strings.NewReader("f"))
	var err error
	var expectedError error = ErrNoFileName
	ted.DoRange()
	if err = ted.DoCommand(); err != expectedError {
		t.Fatalf("expected error '%s', got none", expectedError)
	}
	var path string = "dummy"
	ted.createDummyFile(path)
	defer ted.removeDummyFile(path)
	ted.ReadInput(strings.NewReader("e " + path))
	ted.DoRange()
	if err = ted.DoCommand(); err != nil {
		t.Fatalf("expected no error, got '%s'", err)
	}
	ted.err = &b
	ted.ReadInput(strings.NewReader("f"))
	ted.DoRange()
	if err = ted.DoCommand(); err != nil {
		t.Fatalf("expected no error, got '%s'", err)
	}
	if b.String() != path+"\n" {
		t.Fatalf("expected output to be '%s', got '%s'", path, b.String())
	}
}

// TestCmdGlobal tests the global (g) and inverse global (v) command.
func TestCmdGlobal(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte(",g/hello/d"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"world1", "world2", "world3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte(",g/hello/"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte(",v/hello/"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte(",v/hello/d"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "hello2", "hello3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("2,$g|hello|d"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "world2", "world3"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          []byte("2,$v|hello|d"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "hello2", "hello3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte(",g/hello/s/hello/world/g"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"world1", "world1", "world2", "world2", "world3", "world3"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("3g|hello|"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("g|world|d"),
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "hello2", "hello3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdInsertLines tests the insert (i) command.
func TestCmdInsertLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		data           string
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,3i"),
			data:           "inserted\ntext\n.",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "inserted", "text", "!"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          []byte("1i"),
			data:           "inserted\n.",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"inserted", "hello", "world", "!"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("i"),
			data:           "inserted\n.",
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "inserted", "!"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.in = strings.NewReader(test.data)
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdJoinLines tests the join (j) command.
func TestCmdJoinLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectedError  error
		buffer         []string
		expectedBuffer []string
	}{
		{
			input:          []byte("1,2j"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"helloworld", "!"},
		},
		{
			input:          []byte("2j"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world!"},
		},
		{
			input:          []byte("j"),
			expectedError:  ErrInvalidAddress,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != test.expectedError {
				t.Fatalf("expected error '%s', got %s", test.expectedError, err)
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

// TestCmdMark tests the mark (k) command and the ' address symbol.
func TestCmdMark(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
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
	}
	for _, test := range tests {
		t.Run(string(bytes.Join(test.input, []byte(" "))), func(t *testing.T) {
			for n, cmd := range test.input {
				ted.ReadInput(bytes.NewBuffer(cmd))
				if err := ted.DoRange(); err != nil {
					t.Fatalf("expected no error, got %s", err)
				}
				if err := ted.DoCommand(); err != nil {
					t.Fatalf("expected no error, got %s", err)
				}
				if test.expectedStart[n] != ted.Start {
					t.Fatalf("expected start to be %d, got %d", test.expectedStart[n], ted.Start)
				}
				if test.expectedEnd[n] != ted.End {
					t.Fatalf("expected start to be %d, got %d", test.expectedEnd[n], ted.End)
				}
			}
		})
	}
}

// TestCmdMoveLines tests the movement (m) command.
func TestCmdMoveLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectedError  error
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,2m4"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"!", "this", "hello", "world", "is", "a", "longer", "file"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          []byte("4m6"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "is", "a", "this", "longer", "file"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte("m2"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "file", "world", "!", "this", "is", "a", "longer"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          []byte("m"),
			expectedError:  ErrInvalidCmdSuffix,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  8,
			expectedEnd:    8,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != test.expectedError {
				t.Fatalf("expected error '%s', got %s", test.expectedError, err)
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdPrintLastError tests the print last error (h) command.
func TestCmdPrintLastError(t *testing.T) {
	// TODO: TestCmdPrintLastError
}

// TestCmdPrintTotalLines tests the total lines (=) command.
func TestCmdPrintTotalLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		buffer         []string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,5="),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte("3="),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\n",
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("="),
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
			ted.ReadInput(bytes.NewBuffer(test.input))
			ted.setupTestFile(test.buffer)
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdPrintLines tests all the print commands (p, l, n).
func TestCmdPrintLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		buffer         []string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "1\tA\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A$\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1,5p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A\nB\nC\nD\nE\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte("1,5n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "1\tA\n2\tB\n3\tC\n4\tD\n5\tE\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte("1,5l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A$\nB$\nC$\nD$\nE$\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte(",p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A\nB\nC\nD\nE\nF\nG\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          []byte(",n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "1\tA\n2\tB\n3\tC\n4\tD\n5\tE\n6\tF\n7\tG\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          []byte(",l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "A$\nB$\nC$\nD$\nE$\nF$\nG$\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          []byte("p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "G\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
		{
			input:          []byte("n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectedOutput: "7\tG\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
		{
			input:          []byte("l"),
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
			ted.ReadInput(bytes.NewBuffer(test.input))
			ted.setupTestFile(test.buffer)
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdRead tests the read (r) command.
func TestCmdRead(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	var path string = "dummy"
	ted.createDummyFile(path)
	defer ted.removeDummyFile(path)
	tests := []struct {
		input              []byte
		expectedOutput     string
		expectedStart      int
		expectedEnd        int
		expectedTotalLines int
	}{
		{
			input:              []byte("r " + path),
			expectedOutput:     "52\n",
			expectedStart:      52,
			expectedEnd:        52,
			expectedTotalLines: 52,
		},
		{
			input:              []byte("r !ls main.go"),
			expectedOutput:     "8\n",
			expectedStart:      27,
			expectedEnd:        27,
			expectedTotalLines: 27,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.err = &b
			ted.setupTestFile(dummyFile)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
			if test.expectedTotalLines != len(ted.Lines) {
				t.Fatalf("expected the total lines to be %d, got %d", test.expectedTotalLines, len(ted.Lines))
			}
		})
	}
}

// TestCmdScroll tests the scroll (z) command.
func TestCmdScroll(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	ted.setupTestFile(dummyFile)
	tests := []struct {
		input          []byte
		expectedStart  int
		expectedEnd    int
		expectedOutput string
	}{
		{
			input:          []byte("1,4z5"),
			expectedStart:  9,
			expectedEnd:    9,
			expectedOutput: "D\nE\nF\nG\nH\nI\n",
		},
		{
			input:          []byte("z"),
			expectedStart:  15,
			expectedEnd:    15,
			expectedOutput: "J\nK\nL\nM\nN\nO\n",
		},
		{
			input:          []byte("5z3"),
			expectedStart:  8,
			expectedEnd:    8,
			expectedOutput: "E\nF\nG\nH\n",
		},
		{
			input:          []byte("z20"),
			expectedStart:  26,
			expectedEnd:    26,
			expectedOutput: "I\nJ\nK\nL\nM\nN\nO\nP\nQ\nR\nS\nT\nU\nV\nW\nX\nY\nZ\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.ReadInput(bytes.NewBuffer(test.input))
			ted.out = &b
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
		})
	}
}

// TestCmdSubstitute tests the substitute (s) command.
func TestCmdSubstitute(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte(",s/hello/world"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"world world", "world hello world", "world hello hello world"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte(",s/hello/world/g"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"world world", "world world world", "world world world world"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte(",s/hello/world/2"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello world world", "hello world hello world"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("3s/hello/world/"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "world hello hello world"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("3s/hello/world/g"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "world world world world"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("3s/hello/world/1"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "world hello hello world"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
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
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectedError  error
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,2t3"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "hello", "world", "this", "is", "a", "longer", "file"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("3t4"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "!", "is", "a", "longer", "file"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("t5"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "file", "a", "longer", "file"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte("t"),
			expectedError:  ErrDestinationExpected,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  8,
			expectedEnd:    8,
		},
		{
			input:          []byte("1t"),
			expectedError:  ErrDestinationExpected,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1t0"),
			expectedError:  nil,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  1,
			expectedEnd:    1,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != test.expectedError {
				t.Fatalf("expected error '%s', got %s", test.expectedError, err)
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
			if test.expectedStart != ted.Start {
				t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			}
			if test.expectedEnd != ted.End {
				t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			}
		})
	}
}

// TestCmdUndo tests the undo (u) command.
func TestCmdUndo(*testing.T) {
	// TODO: TestCmdUndo
}

// TestCmdWrite tests the write commands (w, wq, W).
func TestCmdWrite(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	var path string = "dummy"
	if _, err := os.Stat(path); err == nil {
		ted.removeDummyFile(path)
	}
	// TODO: write commands should not change the start and end position.
	tests := []struct {
		input          []byte
		buffer         []string
		expectedOutput string
		expectedStart  int
		expectedEnd    int
		expectedLines  []string
	}{
		{
			input:          []byte("w " + path),
			expectedOutput: "52\n",
			expectedStart:  26,
			expectedEnd:    26,
			expectedLines:  dummyFile,
		},
		{
			input:          []byte("10W " + path),
			expectedOutput: "2\n",
			expectedStart:  26,
			expectedEnd:    26,
			expectedLines:  []string{"J"},
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var b bytes.Buffer
			ted.err = &b
			ted.removeDummyFile(path)
			ted.setupTestFile(dummyFile)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected no error, got %s", err)
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
			// if test.expectedStart != ted.Start {
			// 	t.Fatalf("expected start to be %d, got %d", test.expectedStart, ted.Start)
			// }
			// if test.expectedEnd != ted.End {
			// 	t.Fatalf("expected end to be %d, got %d", test.expectedEnd, ted.End)
			// }
			lines, err := ted.ReadFile(path)
			if err != nil {
				t.Fatalf("expected file '%s' to exist, got error '%s'\n", path, err)
			}
			for i := 0; i < len(lines); i++ {
				if lines[i] != test.expectedLines[i] {
					t.Errorf("expected line %d to be '%s', got '%s'",
						i, test.expectedLines[i], lines[i])
				}
			}
		})
	}
}
