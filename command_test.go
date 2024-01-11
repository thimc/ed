package main

import (
	"bytes"
	"io"
	"log"
	"strings"
	"testing"
)

// TestCmdAppendLines tests the append (a) command.
func TestCmdAppendLines(t *testing.T) {
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		data           string
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,3a"),
			data:           "A\nB\nC\n.\n",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "!", "A", "B", "C"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte("2a"),
			data:           "A\nB\nC\n.\n",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "A", "B", "C", "!"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("a"),
			data:           "appended\n.\n",
			expectError:    false,
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
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		expectedOutput string
	}{
		{
			input:          []byte("!ls *.go | wc -l"), // probably a bad idea
			expectError:    false,
			expectedOutput: "       6\n!\n",
		},
		{
			input:          []byte("!"),
			expectError:    false,
			expectedOutput: "       6\n!\n",
		},
		{
			input:       []byte("!!"),
			expectError: true,
		},
		{
			input:          []byte("! "),
			expectError:    false,
			expectedOutput: "       6\n!\n",
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var err error
			var b bytes.Buffer
			ted.err = &b
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
			}
			if b.String() != test.expectedOutput {
				t.Fatalf("expected output '%s', got '%s'", test.expectedOutput, b.String())
			}
		})
	}
}

// TestCmdChangeLines tests the change (c) command.
func TestCmdChangeLines(t *testing.T) {
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		data           string
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,3c"),
			data:           "changed\ntext\n.\n",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"changed", "text"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          []byte("1c"),
			data:           "changed\n.\n",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"changed", "world", "!"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("c"),
			data:           "changed\n.\n",
			expectError:    false,
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
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,2d"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"!"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("2d"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "!"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		{
			input:          []byte("d"),
			expectError:    true,
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
	log.SetOutput(io.Discard)
	// TODO: TestCmdEdit
}

// TestCmdFile tests the file name (f) command.
func TestCmdFile(t *testing.T) {
	log.SetOutput(io.Discard)
	// TODO: TestCmdFile
}

// TestCmdGlobal tests the global (g) and inverse global (v) command.
func TestCmdGlobal(t *testing.T) {
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte(",g/hello/d"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"world1", "world2", "world3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte(",g/hello/"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte(",v/hello/"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte(",v/hello/d"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "hello2", "hello3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("2,$g|hello|d"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "world2", "world3"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          []byte("2,$v|hello|d"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "hello2", "hello3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte(",g/hello/s/hello/world/g"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"world1", "world1", "world2", "world2", "world3", "world3"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("3g|hello|"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("g|world|d"),
			expectError:    false,
			buffer:         []string{"hello1", "world1", "hello2", "world2", "hello3", "world3"},
			expectedBuffer: []string{"hello1", "hello2", "hello3"},
			expectedStart:  3,
			expectedEnd:    3,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var err error
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		data           string
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,3i"),
			data:           "inserted\ntext\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"hello", "world", "inserted", "text", "!"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          []byte("1i"),
			data:           "inserted\n.",
			expectError:    false,
			buffer:         []string{"hello", "world", "!"},
			expectedBuffer: []string{"inserted", "hello", "world", "!"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("i"),
			data:           "inserted\n.",
			expectError:    false,
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
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
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

// TestCmdMark tests the mark (k) command and the ' address symbol.
func TestCmdMark(t *testing.T) {
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	ted.setupTestFile(dummyFile)
	tests := []struct {
		input         [][]byte
		expectError   []bool
		expectedStart []int
		expectedEnd   []int
	}{
		{
			input:         [][]byte{[]byte("3ka"), []byte("5"), []byte("'a")},
			expectError:   []bool{false, false, false},
			expectedStart: []int{3, 5, 3},
			expectedEnd:   []int{3, 5, 3},
		},
		{
			input:         [][]byte{[]byte("2,5ka"), []byte("'a")},
			expectError:   []bool{false, false},
			expectedStart: []int{2, 5},
			expectedEnd:   []int{5, 5},
		},
	}
	for _, test := range tests {
		t.Run(string(bytes.Join(test.input, []byte(" "))), func(t *testing.T) {
			var err error
			for n, cmd := range test.input {
				ted.ReadInput(bytes.NewBuffer(cmd))
				if err = ted.DoRange(); err != nil && !test.expectError[n] {
					t.Fatalf("expected no error, got %s", err)
				}
				if err = ted.DoCommand(); err != nil && !test.expectError[n] {
					t.Fatalf("expected no error, got %s", err)
				}
				if test.expectError[n] && err == nil {
					t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,2m4"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"!", "this", "hello", "world", "is", "a", "longer", "file"},
			expectedStart:  4,
			expectedEnd:    4,
		},
		{
			input:          []byte("4m6"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "is", "a", "this", "longer", "file"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte("m2"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "file", "world", "!", "this", "is", "a", "longer"},
			expectedStart:  2,
			expectedEnd:    2,
		},
		// {
		// 	input:          []byte("4m"),
		// 	expectError:    true,
		// 	buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
		// 	expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
		// 	expectedStart:  8,
		// 	expectedEnd:    8,
		// },
		{
			input:          []byte("m"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  8,
			expectedEnd:    8,
		},
		// {
		// 	input:          []byte("1,2m1"),
		// 	expectError:    true,
		// 	buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
		// 	expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
		// 	expectedStart:  8,
		// 	expectedEnd:    8,
		// },
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var err error
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))

			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	// TODO: TestCmdPrintLastError
}

// TestCmdPrintTotalLines tests the total lines (=) command.
func TestCmdPrintTotalLines(t *testing.T) {
	log.SetOutput(io.Discard)
	// TODO: TestCmdPrintTotalLines
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		buffer         []string
		expectError    bool
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,5="),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "7\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte("3="),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "7\n",
			expectedStart:  3,
			expectedEnd:    3,
		},
		{
			input:          []byte("="),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "7\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var err error
			var b bytes.Buffer
			ted.out = &b
			ted.ReadInput(bytes.NewBuffer(test.input))
			ted.setupTestFile(test.buffer)
			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		buffer         []string
		expectError    bool
		expectedOutput string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "A\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "1\tA\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "A$\n",
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1,5p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "A\nB\nC\nD\nE\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte("1,5n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "1\tA\n2\tB\n3\tC\n4\tD\n5\tE\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte("1,5l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "A$\nB$\nC$\nD$\nE$\n",
			expectedStart:  1,
			expectedEnd:    5,
		},
		{
			input:          []byte(",p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "A\nB\nC\nD\nE\nF\nG\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          []byte(",n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "1\tA\n2\tB\n3\tC\n4\tD\n5\tE\n6\tF\n7\tG\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          []byte(",l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "A$\nB$\nC$\nD$\nE$\nF$\nG$\n",
			expectedStart:  1,
			expectedEnd:    7,
		},
		{
			input:          []byte("p"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "G\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
		{
			input:          []byte("n"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "7\tG\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
		{
			input:          []byte("l"),
			buffer:         []string{"A", "B", "C", "D", "E", "F", "G"},
			expectError:    false,
			expectedOutput: "G$\n",
			expectedStart:  7,
			expectedEnd:    7,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var err error
			var b bytes.Buffer
			ted.out = &b
			ted.ReadInput(bytes.NewBuffer(test.input))
			ted.setupTestFile(test.buffer)
			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	// TODO: TestCmdRead
}

// TestCmdScroll tests the scroll (z) command.
func TestCmdScroll(t *testing.T) {
	log.SetOutput(io.Discard)
	// TODO: TestCmdScroll
}

// TestCmdSubstitute tests the substitute (s) command.
func TestCmdSubstitute(t *testing.T) {
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
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
			var err error
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	// TODO: TestCmdToggleError
}

// TestCmdTogglePrompt tests the toggle prompt (P) command.
func TestCmdTogglePrompt(t *testing.T) {
	log.SetOutput(io.Discard)
	// TODO: TestCmdTogglePrompt
}

// TestCmdTransferLines tests the transfer (t) command.
func TestCmdTransferLines(t *testing.T) {
	log.SetOutput(io.Discard)
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedStart  int
		expectedEnd    int
	}{
		{
			input:          []byte("1,2t3"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "hello", "world", "this", "is", "a", "longer", "file"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("3t4"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "!", "is", "a", "longer", "file"},
			expectedStart:  5,
			expectedEnd:    5,
		},
		{
			input:          []byte("t5"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "file", "a", "longer", "file"},
			expectedStart:  6,
			expectedEnd:    6,
		},
		{
			input:          []byte("t"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  8,
			expectedEnd:    8,
		},
		{
			input:          []byte("1t"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  1,
			expectedEnd:    1,
		},
		{
			input:          []byte("1t0"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedStart:  1,
			expectedEnd:    1,
		},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			var err error
			ted.setupTestFile(test.buffer)
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err = ted.DoRange(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if err = ted.DoCommand(); err != nil && !test.expectError {
				t.Fatalf("expected no error, got %s", err)
			}
			if test.expectError && err == nil {
				t.Fatalf("expected error, got none")
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
	log.SetOutput(io.Discard)
	// TODO: TestCmdUndo
}

// TestCmdWrite tests the write commands (w, wq, W).
func TestCmdWrite(t *testing.T) {
	log.SetOutput(io.Discard)
	// TODO: TestCmdWrite
}
