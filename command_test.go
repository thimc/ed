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

// TestCmdBang tests the ! command.
func TestCmdBang(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
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

// TestCmdDeleteLines tests the (d)elete command.
func TestCmdDeleteLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
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

// TestCmdGlobal tests the (g)lobal and in(v)erse global command.
func TestCmdGlobal(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
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

// TestCmdInsertLines tests the (i)nsert command.
func TestCmdInsertLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
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

// TestCmdMark tests the mark (k) command and the ' address symbol.
func TestCmdMark(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
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

// TestCmdMoveLines tests the (m)ovement command.
func TestCmdMoveLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedDot    int
	}{
		{
			input:          []byte("1,2m4"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"!", "this", "hello", "world", "is", "a", "longer", "file"},
			expectedDot:    4,
		},
		{
			input:          []byte("4m6"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "is", "a", "this", "longer", "file"},
			expectedDot:    6,
		},
		{
			input:          []byte("m2"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "file", "world", "!", "this", "is", "a", "longer"},
			expectedDot:    3,
		},
		{
			input:          []byte("4m"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedDot:    6,
		},
		{
			input:          []byte("m"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedDot:    6,
		},
		{
			input:          []byte("1,2m1"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedDot:    8,
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
		})
	}
}

// TestCmdSubstitute tests the substitute (s) command.
func TestCmdSubstitute(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedDot    int
	}{
		{
			input:          []byte(",s"),
			expectError:    true,
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedDot:    3,
		},
		{
			input:          []byte(",s/hello/world"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"world world", "world hello world", "world hello hello world"},
			expectedDot:    3,
		},
		{
			input:          []byte(",s/hello/world/g"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"world world", "world world world", "world world world world"},
			expectedDot:    3,
		},
		{
			input:          []byte(",s/hello/world/2"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello world world", "hello world hello world"},
			expectedDot:    3,
		},
		{
			input:          []byte("3s/hello/world/"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "world hello hello world"},
			expectedDot:    3,
		},
		{
			input:          []byte("3s/hello/world/g"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "world world world world"},
			expectedDot:    3,
		},
		{
			input:          []byte("3s/hello/world/1"),
			buffer:         []string{"hello world", "hello hello world", "hello hello hello world"},
			expectedBuffer: []string{"hello world", "hello hello world", "world hello hello world"},
			expectedDot:    3,
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
		})
	}
}

// TestCmdTransferLines tests the (t)ransfer command.
func TestCmdTransferLines(t *testing.T) {
	var ted *Editor = NewEditor(nil, io.Discard, io.Discard)
	log.SetOutput(io.Discard)
	tests := []struct {
		input          []byte
		expectError    bool
		buffer         []string
		expectedBuffer []string
		expectedDot    int
	}{
		{
			input:          []byte("1,2t3"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "hello", "world", "this", "is", "a", "longer", "file"},
			expectedDot:    5,
		},
		{
			input:          []byte("3t4"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "!", "is", "a", "longer", "file"},
			expectedDot:    5,
		},
		{
			input:          []byte("t5"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "file", "a", "longer", "file"},
			expectedDot:    6,
		},
		{
			input:          []byte("t"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedDot:    8,
		},
		{
			input:          []byte("1t"),
			expectError:    true,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedDot:    1,
		},
		{
			input:          []byte("1t0"),
			expectError:    false,
			buffer:         []string{"hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedBuffer: []string{"hello", "hello", "world", "!", "this", "is", "a", "longer", "file"},
			expectedDot:    1,
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
			if ted.Dot != test.expectedDot {
				t.Fatalf("expected the dot to be %d, got %d", test.expectedDot, ted.Dot)
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
