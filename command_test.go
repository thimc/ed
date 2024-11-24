package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestCmdAppend(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd            string
		buffer         []string
		init           position
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "a\n1\n.",
			buffer:         []string{"A", "B", "C", "D", "E"},
			expectedBuffer: []string{"A", "B", "C", "D", "E", "1"},
			init:           position{start: 5, end: 5, dot: 5},
			expect:         position{start: 5, end: 5, dot: 6, addrc: 0},
		},
		{
			cmd:            "2a\n1\n.",
			buffer:         []string{"A", "B", "C", "D", "E"},
			expectedBuffer: []string{"A", "B", "1", "C", "D", "E"},
			expect:         position{start: 2, end: 2, dot: 3, addrc: 1},
		},
		{
			cmd:            "1,3a\nX\nY\nZ\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "C", "X", "Y", "Z"},
			init:           position{start: 3, end: 3, dot: 3, addrc: 0},
			expect:         position{start: 1, end: 3, dot: 6, addrc: 2},
		},
		{
			cmd:            "0a\nA\n.\n",
			buffer:         []string{},
			expectedBuffer: []string{"A"},
			expect:         position{start: 0, end: 0, dot: 1, addrc: 0},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			setPosition(ted, tt.init)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
		})
	}
}

func TestCmdChange(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd            string
		buffer         []string
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "c\nZ\n.\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"A", "B", "C", "D", "E", "Z"},
			expect:         position{start: 6, end: 6, dot: 6, addrc: 0},
		},
		{
			cmd:            "2c\ntest\n.\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"A", "test", "C", "D", "E", "F"},
			expect:         position{start: 2, end: 2, dot: 2, addrc: 1},
		},
		{
			cmd:            "1,3c\nX\nY\nZ\n.\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"X", "Y", "Z", "D", "E", "F"},
			expect:         position{start: 1, end: 3, dot: 3, addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
		})
	}

}

func TestCmdDelete(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd            string
		buffer         []string
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "2d\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"A", "C", "D", "E", "F"},
			expect:         position{start: 2, end: 2, dot: 2, addrc: 1},
		},
		{
			cmd:            "3,d\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"A", "B", "D", "E", "F"},
			expect:         position{start: 3, end: 3, dot: 3, addrc: 1},
		},
		{
			cmd:            "2,4d\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"A", "E", "F"},
			expect:         position{start: 2, end: 4, dot: 2, addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
		})
	}
}

func TestCmdEdit(t *testing.T) {
	var path = "dummy"
	createDummyFile(path, t)
	defer os.Remove(path)
	tests := []struct {
		cmd            string
		expect         position
		buffer         []string
		expectedBuffer []string
		expectedOutput string
		err            error
	}{
		{
			cmd:            "e !ls ed.go\n",
			expect:         position{start: 0, end: 0, dot: 1, addrc: 0},
			buffer:         nil,
			expectedBuffer: []string{"ed.go"},
			expectedOutput: "6\n",
		},
		{
			cmd:            "e " + path + "\n",
			expect:         position{start: 0, end: 0, dot: len(dummyFile), addrc: 0},
			buffer:         nil,
			expectedBuffer: dummyFile,
			expectedOutput: "52\n",
		},
		{
			cmd:            "5e\n",
			expect:         position{start: 5, end: 5, dot: len(dummyFile), addrc: 1},
			buffer:         dummyFile,
			expectedBuffer: dummyFile,
			err:            ErrUnexpectedAddress,
		},
		{cmd: "e\n", buffer: nil, expectedBuffer: nil, err: ErrNoFileName},
		{cmd: "e|\n", buffer: nil, expectedBuffer: nil, err: ErrUnexpectedCmdSuffix},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdin(strings.NewReader(tt.cmd)), WithStdout(&b), WithStderr(&b))
			)
			ted.printErrors = true
			if tt.buffer != nil {
				setupMemoryFile(ted, tt.buffer)
			}
			if err := ted.Do(); err != tt.err {
				if tt.err != nil {
					t.Fatalf("expected error %q, got %q", tt.err, err)
				}
				t.Fatal(err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
			if b.String() != tt.expectedOutput {
				t.Fatalf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdFile(t *testing.T) {
	var path = "dummy"
	createDummyFile(path, t)
	defer os.Remove(path)
	tests := []struct {
		cmd              string
		err              error
		filename         string
		expectedFilename string
		expectedOutput   string
	}{
		{
			cmd:              "f filename\n",
			filename:         "dummy",
			expectedFilename: "filename",
			expectedOutput:   "filename\n",
		},
		{
			cmd:      "f\n",
			filename: "",
			err:      ErrNoFileName,
		},
		{
			cmd: "2f\n",
			err: ErrUnexpectedAddress,
		},
		{
			cmd: "f !ls\n",
			err: ErrInvalidRedirection,
		},
		{
			cmd: "f!\n",
			err: ErrUnexpectedCmdSuffix,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdin(strings.NewReader(tt.cmd)), WithStdout(&b), WithStderr(&b))
			)
			ted.printErrors = true
			setupMemoryFile(ted, dummyFile)
			ted.path = tt.filename
			if err := ted.Do(); err != tt.err {
				if tt.err != nil {
					t.Fatalf("expected error %q, got %q", tt.err, err)
				}
				t.Fatal(err)
			}
			if ted.path != tt.expectedFilename {
				t.Fatalf("expected buffer name to be %q, got %q", tt.expectedFilename, ted.path)
			}
			if b.String() != tt.expectedOutput {
				t.Fatalf("expected output to be %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdGlobal(t *testing.T) {
	var (
		buffer = dummyFile[:5]
		last   = len(buffer)
	)
	tests := []struct {
		cmd            string
		init           position
		buffer         []string
		expect         position
		expectedBuffer []string
		expectedOutput string
		err            error
	}{
		{
			cmd:            "g/B/dn",
			expect:         position{start: 2, end: 2, dot: 2, addrc: 0},
			expectedBuffer: []string{"A", "C", "D", "E"},
			expectedOutput: "2\tC\n",
		},
		{
			cmd:            "g/Z/\n",
			expect:         position{start: 1, end: last, dot: last, addrc: 0},
			expectedBuffer: buffer,
		},
		{
			cmd:            "g/B/d\n",
			expect:         position{start: 2, end: 2, dot: 2, addrc: 0},
			expectedBuffer: []string{"A", "C", "D", "E"},
		},
		{
			cmd:            "v/B/d\n",
			expect:         position{start: 2, end: 2, dot: 1, addrc: 0},
			expectedBuffer: []string{"B"},
		},
		{
			cmd:            "G/B\n\n",
			expect:         position{start: 1, end: 5, dot: 2, addrc: 0},
			expectedBuffer: buffer,
			expectedOutput: "B\n",
		},
		{
			cmd:            "g/yes/d\\\np",
			buffer:         []string{"A yes", "a no", "B yes", "b no", "C yes", "c no"},
			expect:         position{start: 3, end: 3, dot: 3, addrc: 0},
			expectedBuffer: []string{"a no", "b no", "c no"},
			expectedOutput: "a no\nb no\nc no\n",
		},
		{
			cmd:            "g/A/\\",
			expect:         position{start: 1, end: 5, dot: 5, addrc: 0},
			expectedBuffer: buffer,
			err:            ErrUnexpectedEOF,
		},
		{
			cmd:            "g/A/pZ\n",
			expect:         position{start: 1, end: 1, dot: 1, addrc: 0},
			expectedBuffer: buffer,
			err:            ErrInvalidCmdSuffix,
		},
		{
			cmd:            "g/A/Z\n",
			expect:         position{start: 1, end: 1, dot: 1, addrc: 0},
			expectedBuffer: buffer,
			err:            ErrUnknownCmd,
		},
		{
			cmd:            "g\n",
			expect:         position{start: 1, end: 5, dot: 5, addrc: 0},
			expectedBuffer: buffer,
			err:            ErrInvalidPatternDelim,
		},
		{
			cmd:            "g/\n",
			expect:         position{start: 1, end: 5, dot: 5, addrc: 0},
			expectedBuffer: buffer,
			err:            ErrNoPrevPattern,
		},
		{
			cmd:            "g/A/g\n",
			expect:         position{start: 1, end: 1, dot: 1, addrc: 0},
			expectedBuffer: buffer,
			err:            ErrCannotNestGlobal,
		},
		{
			cmd:            "G/B/\n&\n",
			expect:         position{start: 1, end: 5, dot: 2, addrc: 0},
			expectedBuffer: buffer,
			expectedOutput: "B\n",
			err:            ErrNoPreviousCmd,
		},
	}
	_ = last // XXX
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdout(&b), WithStderr(io.Discard))
				buf = buffer
			)
			if tt.buffer != nil {
				buf = tt.buffer
			}
			setupMemoryFile(ted, buf)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Errorf("expected buffer %q, got %q", tt.expectedBuffer, ted.lines)
			}
			if b.String() != tt.expectedOutput {
				t.Fatalf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdHelp(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	ted.printErrors = true
	setupMemoryFile(ted, dummyFile)
	tests := []struct {
		cmd    string
		output string
		err    error
	}{
		{cmd: "2h\n", err: ErrUnexpectedAddress},
		{cmd: "h}\n", err: ErrInvalidCmdSuffix},
		{cmd: "h\n", err: explainError{ErrInvalidCmdSuffix}},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
		})
	}
}

func TestCmdToggleHelp(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	tests := []struct {
		cmd    string
		output string
		err    error
	}{
		{cmd: "2H\n", err: ErrDefault},
		{cmd: "H\n", err: explainError{ErrUnexpectedAddress}},
		{cmd: "3H\n", err: ErrUnexpectedAddress},
		{cmd: "H=\n", err: ErrInvalidCmdSuffix},
	}
	setupMemoryFile(ted, dummyFile)
	ted.printErrors = false
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
		})
	}
}

func TestCmdInsert(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd, input     string
		buffer         []string
		init           position
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "1,3i\n",
			input:          "X\nY\nZ\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "X", "Y", "Z", "C"},
			init:           position{start: 3, end: 3, dot: 3, addrc: 0},
			expect:         position{start: 1, end: 3, dot: 5, addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd + tt.input)
			ted.tokenizer = nil
			setPosition(ted, tt.init)
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
		})
	}
}

func TestCmdJoin(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd, input     string
		buffer         []string
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "1,3j",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"ABC"},
			expect:         position{start: 1, end: 3, dot: 1, addrc: 2},
		},
		{
			cmd:            "2,4j",
			buffer:         []string{"A", "B", "C", "D", "E"},
			expectedBuffer: []string{"A", "BCD", "E"},
			expect:         position{start: 2, end: 4, dot: 2, addrc: 2},
		},
		{
			cmd:            ",j",
			buffer:         []string{"A", "B", "C", "D", "E"},
			expectedBuffer: []string{"ABCDE"},
			expect:         position{start: 1, end: 5, dot: 1, addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd + "\n")
			ted.tokenizer = nil
			if err := ted.Do(); err != nil {
				t.Fatalf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
		})
	}
}

func TestCmdMark(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd    string
		mark   rune
		buffer []string
		init   position
		expect position
	}{
		{
			cmd:    "k",
			mark:   'a',
			buffer: []string{"A", "B", "C", "D", "E"},
			init:   position{start: 3, end: 3, dot: 5},
			expect: position{start: 3, end: 3, dot: 5, addrc: 1},
		},
	}
	for _, tt := range tests {
		cmd := fmt.Sprint(tt.init.start) + tt.cmd + string(tt.mark) + "\n"
		t.Run(cmd, func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != nil {
				t.Errorf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			markpos := ted.mark[tt.mark-'a']
			if markpos != tt.init.end {
				t.Fatalf("expected mark %c to point at %d, got %d", tt.mark, tt.init.end, markpos)
			}
		})
	}
}

func TestCmdListNumberPrint(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	tests := []struct {
		cmd    string
		buffer []string
		init   position
		expect position
	}{
		{
			cmd:    "2,5l\n",
			buffer: dummyFile,
			init:   position{start: 3, end: 3, dot: 3},
			expect: position{start: 2, end: 5, dot: 5, addrc: 2},
		},
		{
			cmd:    ",n\n",
			buffer: dummyFile,
			expect: position{start: 1, end: len(dummyFile), dot: len(dummyFile), addrc: 2},
		},
		{
			cmd:    ";p\n",
			buffer: dummyFile,
			init:   position{start: 1, end: 1, dot: 1},
			expect: position{start: 1, end: len(dummyFile), dot: len(dummyFile), addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd + "\n")
			ted.tokenizer = nil
			if err := ted.Do(); err != nil {
				t.Errorf("expected no error, got %q", err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if b.String() == "" {
				t.Fatalf("expected non-empty buffer")
			}
		})
	}
}

func TestCmdMove(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	tests := []struct {
		cmd            string
		init           position
		buffer         []string
		expect         position
		expectedBuffer string
		err            error
	}{
		{
			cmd:            "1,5m9\n",
			init:           position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			buffer:         dummyFile,
			expect:         position{start: 1, end: 5, dot: 9, addrc: 1},
			expectedBuffer: "F\nG\nH\nI\nA\nB\nC\nD\nE\nJ\nK\nL\nM\nN\nO\nP\nQ\nR\nS\nT\nU\nV\nW\nX\nY\nZ",
		},
		{
			cmd:            "1,5mz\n",
			buffer:         dummyFile,
			expectedBuffer: strings.Join(dummyFile, "\n"),
			err:            ErrDestinationExpected,
		},
		{
			cmd:            "m1\n",
			buffer:         dummyFile,
			expectedBuffer: strings.Join(dummyFile, "\n"),
			err:            ErrInvalidAddress,
		},
		{
			cmd:            "1,2m1\n",
			buffer:         dummyFile,
			expect:         position{start: 1, end: 2, dot: 0, addrc: 1},
			expectedBuffer: strings.Join(dummyFile, "\n"),
			err:            ErrInvalidDestination,
		},
		{
			cmd:            "1m1a\n",
			buffer:         dummyFile,
			expect:         position{start: 1, end: 1, dot: 0, addrc: 1},
			expectedBuffer: strings.Join(dummyFile, "\n"),
			err:            ErrInvalidCmdSuffix,
		},
		{
			cmd:            "m\n",
			buffer:         []string{""},
			expectedBuffer: "",
			err:            ErrInvalidAddress,
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Errorf("expected error %q, got %q", tt.err, err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			expect := strings.Split(tt.expectedBuffer, "\n")
			if !reflect.DeepEqual(ted.lines, expect) {
				t.Fatalf("expected buffer %q, got %q", expect, ted.lines)
			}
		})
	}
}

func TestCmdPrompt(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(io.Discard), WithPrompt(""))
	)
	tests := []struct {
		cmd    string
		err    error
		output string
		expect bool
	}{
		{cmd: "P\n", err: nil, expect: true},
		{cmd: "P\n", err: nil, output: DefaultPrompt, expect: false},
		{cmd: "1,2P\n", err: ErrUnexpectedAddress},
		{cmd: "P\n", expect: true},
		{cmd: "P\n", output: DefaultPrompt, expect: false},
		{cmd: "P#\n", err: ErrInvalidCmdSuffix},
		{cmd: "Pp\n", output: dummyFile[len(dummyFile)-1] + "\n", expect: true},
		{cmd: "P\n", output: DefaultPrompt, expect: false},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, dummyFile)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			if ted.showPrompt != tt.expect {
				t.Fatalf("expected show prompt to be %t, got %t", tt.expect, ted.showPrompt)
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
		})
	}
}

func TestCmdQuit(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	ted.printErrors = true
	setupMemoryFile(ted, dummyFile)
	tests := []struct {
		cmd      string
		modified bool
		err      error
	}{
		{cmd: "1q\n", err: ErrUnexpectedAddress},
		{
			cmd:      "q\n",
			modified: true,
			err:      ErrFileModified,
		},
		{
			cmd:      "Q\n",
			modified: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
		})
		ted.modified = tt.modified
		defer func() {
			recover()
		}()
		if err := ted.Do(); err != tt.err {
			t.Fatalf("expected error %q, got %q", tt.err, err)
		}
	}
}

func TestCmdRead(t *testing.T) {
	var (
		b    bytes.Buffer
		ted  = New(WithStdout(&b), WithStderr(&b))
		path = "dummy_read"
	)
	ted.printErrors = true
	createDummyFile(path, t)
	defer os.Remove(path)
	tests := []struct {
		cmd    string
		buffer []string
		output string
		err    error
		setup  bool
	}{
		{
			cmd:    "r " + path + "\n",
			buffer: dummyFile,
			output: fmt.Sprint(len(dummyFile)*2) + "\n",
		},
		{
			cmd:    "r\n",
			buffer: dummyFile,
			output: fmt.Sprint(len(dummyFile)*2) + "\n",
			setup:  true,
		},
		{cmd: "r#\n", buffer: dummyFile, err: ErrUnexpectedCmdSuffix},
		{cmd: "r\n", buffer: dummyFile, err: ErrNoFileName},
		{cmd: "r non-existing-file\n", buffer: dummyFile, err: ErrCannotOpenFile},
		{cmd: "r  twospaces\n", buffer: dummyFile, err: ErrInvalidFileName},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if tt.err != nil {
				ted.path = ""
			}
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
			if !reflect.DeepEqual(tt.buffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.buffer, ted.lines)
			}
		})
	}
}

func TestCmdSubstitute(t *testing.T) {
	// TODO(thimc): Add tests cases for the the substitute command
	// with `r`, `p`, `l` and `n` command suffixes.
	// TODO(thimc): Add tests cases for the substitute command but
	// reusing the last search criteria and `%` (last replacement
	// string) as replacement text.
	var (
		b      bytes.Buffer
		buffer = []string{
			"A A A A A",
			"A A A A A",
			"B B B B B",
			"B B B B B",
			"C C C C C",
			"C C C C C",
			"D D D D D",
			"D D D D D",
		}
		last = len(buffer)
	)
	tests := []struct {
		cmd               []string
		expect            position
		expectedOutput    string
		buffer            []string
		expectedBuffer    []string
		err               []error
		expectedSubSuffix subSuffix
		expectedCmdSuffix cmdSuffix
	}{
		{
			cmd:               []string{",s/A/X/gp\n"},
			expectedBuffer:    []string{"X X X X X", "X X X X X", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:            position{start: 1, end: last, dot: 2, addrc: 2},
			expectedOutput:    "X X X X X\n",
			expectedCmdSuffix: cmdSuffixPrint,
			err:               []error{nil},
		},
		{
			cmd:            []string{",s/A/X/g\n"},
			expectedBuffer: []string{"X X X X X", "X X X X X", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: last, dot: 2, addrc: 2},
			err:            []error{nil},
		},
		{
			cmd:            []string{"1s/A/X/\n"},
			expectedBuffer: []string{"X A A A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
			err:            []error{nil},
		},
		{
			cmd:            []string{"1s/A/X/g\n"},
			expectedBuffer: []string{"X X X X X", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
			err:            []error{nil},
		},
		{
			cmd:            []string{"1s/A/X/3\n"},
			expectedBuffer: []string{"A A X A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
			err:            []error{nil},
		},
		{
			cmd:            []string{"1s/A/&X/3\n"},
			expectedBuffer: []string{"A A AX A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
			err:            []error{nil},
		},
		{
			cmd:            []string{`3,5s/ (.)(.)/_\2_\1X\2_/` + "\n"},
			expectedBuffer: []string{"A A A A A", "A A A A A", "B_ _BX _B B B", "B_ _BX _B B B", "C_ _CX _C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 3, end: 5, dot: 5, addrc: 2},
			err:            []error{nil},
		},
		{
			cmd:               []string{",s/A/z/p\n", ",s\n"},
			expectedBuffer:    []string{"z z A A A", "z z A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:            position{start: 1, end: 8, dot: 2, addrc: 2},
			expectedSubSuffix: subRepeat,
			expectedOutput:    "z A A A A\n",
			err:               []error{nil, nil},
		},
		{
			cmd:            []string{",s/X/Y/g\n"},
			expectedBuffer: buffer,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            []error{ErrNoMatch},
		},
		{
			cmd:            []string{",s//Y/\n"},
			expectedBuffer: buffer,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            []error{ErrNoPrevPattern},
		},
		{
			cmd:            []string{",s/X/%/\n"},
			expectedBuffer: buffer,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            []error{ErrNoPreviousSub},
		},
		{
			cmd:               []string{",sgpr\n"},
			expectedBuffer:    buffer,
			expect:            position{start: 1, end: last, dot: last, addrc: 2},
			err:               []error{ErrNoPrevPattern},
			expectedSubSuffix: subRepeat | subGlobal | subPrint | subLastRegex,
		},
		{
			cmd:               []string{"s/.*/some/nl\n", ",sp}\n"},
			expectedBuffer:    append(append([]string{}, buffer[:len(buffer)-1]...), "some"),
			expectedOutput:    "8\tsome$\n",
			expect:            position{start: 1, end: last, dot: 8, addrc: 2},
			err:               []error{nil, ErrInvalidCmdSuffix},
			expectedSubSuffix: subPrint,
		},
		{
			cmd:            []string{"s/.*/some/p\n", ",s/.*/%/\n"},
			expectedBuffer: []string{"some", "some", "some", "some", "some", "some", "some", "some"},
			expectedOutput: "some\n",
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            []error{nil, nil},
		},
		{
			cmd:               []string{",s\n"},
			expectedBuffer:    buffer,
			expect:            position{start: 1, end: last, dot: last, addrc: 2},
			err:               []error{ErrNoPrevPattern},
			expectedSubSuffix: subRepeat,
		},
		{
			cmd:            []string{",s a z p\n"},
			expectedBuffer: buffer,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            []error{ErrInvalidPatternDelim},
		},
		{
			cmd:            []string{",s/A\n"},
			expectedBuffer: []string{" A A A A", " A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: last, dot: 2, addrc: 2},
			err:            []error{nil},
		},
		{
			cmd:               []string{"s/#[[:alnum:]_]*/test/2\n", "s2\n"},
			expect:            position{start: 1, end: 1, dot: 1},
			buffer:            []string{"hello #foo #bar #baz world"},
			expectedBuffer:    []string{"hello #foo test test world"},
			expectedSubSuffix: subRepeat | subNth,
			err:               []error{nil, nil},
		},
		{
			cmd:               []string{",s/A/Z/p\n"},
			expectedBuffer:    []string{"Z A A A A", "Z A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:            position{start: 1, end: last, dot: 2, addrc: 2},
			expectedOutput:    "Z A A A A\n",
			expectedCmdSuffix: cmdSuffixPrint,
			err:               []error{nil},
		},
		{
			cmd:               []string{",s/A/B/p/\n"},
			expectedBuffer:    buffer,
			expect:            position{start: 1, end: last, dot: last, addrc: 2},
			err:               []error{ErrInvalidCmdSuffix},
			expectedCmdSuffix: cmdSuffixPrint,
		},
		{
			cmd:               []string{",s/^A.*/&/gp\n"},
			expectedBuffer:    buffer,
			expectedOutput:    "A A A A A\n",
			expect:            position{start: 1, end: last, dot: 2, addrc: 2},
			expectedCmdSuffix: cmdSuffixPrint,
			err:               []error{nil},
		},
	}
	for _, tt := range tests {
		t.Run(string(strings.Join(tt.cmd, "\n")), func(t *testing.T) {
			b.Reset()
			var (
				ted = New(WithStdout(&b), WithStderr(io.Discard))
				buf = buffer
			)
			if tt.buffer != nil {
				buf = tt.buffer
			}
			setupMemoryFile(ted, buf)
			for n, cmd := range tt.cmd {
				ted.in = strings.NewReader(cmd)
				ted.tokenizer = nil
				if err := ted.Do(); err != tt.err[n] {
					t.Fatalf("expected error %q, got %q", tt.err[n], err)
				}
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(tt.expectedBuffer, ted.lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.lines)
			}
			if b.String() != tt.expectedOutput {
				t.Fatalf("expected output: %q, got %q", tt.expectedOutput, b.String())
			}
			if ted.ss != tt.expectedSubSuffix {
				t.Fatalf("expected substitution flags %d, got %d", tt.expectedSubSuffix, ted.ss)
			}
			if ted.cs != tt.expectedCmdSuffix {
				t.Fatalf("expected command flags %d, got %d", tt.expectedCmdSuffix, ted.cs)
			}
		})
	}
}

func TestCmdScroll(t *testing.T) {
	tests := []struct {
		cmd            string
		buffer         []string
		init           position
		expect         position
		expectedOutput string
		err            error
	}{
		{
			cmd:            "2z6\n",
			buffer:         dummyFile,
			init:           position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:         position{start: 1, end: 2, dot: 8, addrc: 1},
			expectedOutput: strings.Join(dummyFile[1:8], "\n") + "\n",
		},
		{
			cmd:    "2z1234567891011121314151617181920\n",
			buffer: dummyFile,
			init:   position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect: position{start: 1, end: 2, dot: len(dummyFile), addrc: 1},
			err:    ErrNumberOutOfRange,
		},
		{
			cmd:    "z\n",
			expect: position{start: 1, end: 1, dot: 0, addrc: 0},
			err:    ErrInvalidAddress,
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdout(&b), WithStderr(&b))
			)
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Errorf("expected error %q, got %q", tt.err, err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if b.String() != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdTransfer(t *testing.T) {
	var (
		b    bytes.Buffer
		ted  = New(WithStdout(&b), WithStderr(&b))
		last = len(dummyFile)
	)
	tests := []struct {
		cmd            string
		buffer         []string
		expect         position
		expectedBuffer []string
		err            error
	}{
		{
			cmd:            "1,5t3\n",
			buffer:         dummyFile,
			expect:         position{start: 1, end: 5, dot: 8, addrc: 1},
			expectedBuffer: []string{"A", "B", "C", "A", "B", "C", "D", "E", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"},
		},
		{
			cmd:            fmt.Sprint(last+2) + "t5\n",
			buffer:         dummyFile,
			expect:         position{start: last, end: last, dot: last, addrc: 1},
			expectedBuffer: dummyFile,
			err:            ErrInvalidAddress,
		},
		{
			cmd:            "1,5tz\n",
			buffer:         dummyFile,
			expect:         position{start: last, end: last, dot: last, addrc: 0},
			expectedBuffer: dummyFile,
			err:            ErrDestinationExpected,
		},
		{
			cmd:            "t5}\n",
			buffer:         dummyFile,
			expect:         position{start: last, end: last, dot: last, addrc: 1},
			expectedBuffer: dummyFile,
			err:            ErrInvalidCmdSuffix,
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Errorf("expected error %q, got %q", tt.err, err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(ted.lines, tt.expectedBuffer) {
				t.Fatalf("expected buffer %q, got %q", tt.expectedBuffer, ted.lines)
			}
		})
	}
}

func TestCmdUndo(t *testing.T) {
	var (
		buffer = dummyFile
		last   = len(buffer)
	)
	tests := []struct {
		cmds           []string
		init           position
		expect         position
		expectedOutput string
		err            error
	}{
		{
			cmds:   []string{",d\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 1, end: 1, dot: last},
		},
		{
			cmds:   []string{"2,4d\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 2, end: 2, dot: last},
		},
		{
			cmds:   []string{"1d\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 1, end: 1, dot: last},
		},
		{
			cmds:   []string{"1,3a\ntest\ntest2\n.\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 5, end: 5, dot: last},
		},
		{
			cmds:   []string{"2,3c\ntest\ntest2\n.\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 3, end: 3, dot: last},
		},
		{
			cmds:   []string{"4,9j\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 4, end: 4, dot: last},
		},
		{
			cmds:   []string{"3,5m10\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 10, end: 10, dot: last},
		},
		{
			cmds:   []string{"2,6t12\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 17, end: 17, dot: last},
		},
		{
			cmds:   []string{",s/D/test\n", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 4, end: 4, dot: last},
		},
		{
			cmds:   []string{"5u\n"},
			expect: position{start: 5, end: 5, dot: 0, addrc: 1},
			err:    ErrUnexpectedAddress,
		},
		{
			cmds: []string{"u\n"},
			err:  ErrNothingToUndo,
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.cmds, "\n"), func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdout(&b), WithStderr(&b))
			)
			setupMemoryFile(ted, buffer)
			setPosition(ted, tt.init)
			for _, cmd := range tt.cmds {
				ted.in = strings.NewReader(cmd)
				ted.tokenizer = nil
				if err := ted.Do(); err != tt.err {
					t.Errorf("expected error %q, got %q", tt.err, err)
				}
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if !reflect.DeepEqual(ted.lines, dummyFile) {
				t.Errorf("expected buffer %q, got %q", dummyFile, ted.lines)
			}
			if b.String() != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdWrite(t *testing.T) {
	// TODO(thimc): Add test cases for the write command with an
	// unexpected command suffix and a test where no path is provided.
	// Also add tests for 'wq' and 'Wq'.
	var path = "dummy_write"
	os.Remove(path)
	defer os.Remove(path)
	tests := []struct {
		cmd             []string
		buffer          []string
		init            position
		expect          position
		expectedOutput  string
		expectedContent string
		err             error
	}{
		{
			cmd:             []string{"1w " + path + "\n"},
			buffer:          dummyFile,
			init:            position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:          position{start: 1, end: 1, dot: len(dummyFile), addrc: 1},
			expectedOutput:  "2\n",
			expectedContent: "A\n",
		},
		{
			cmd:             []string{"1w " + path + "\n", "2,3W " + path + "\n"},
			buffer:          dummyFile,
			init:            position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:          position{start: 2, end: 3, dot: len(dummyFile), addrc: 2},
			expectedOutput:  "2\n4\n",
			expectedContent: "A\nB\nC\n",
		},
		{
			cmd:             []string{"1w " + path + "\n", "f " + path + "\n", "2,3W\n"},
			buffer:          dummyFile,
			init:            position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:          position{start: 2, end: 3, dot: len(dummyFile), addrc: 2},
			expectedOutput:  "2\n" + path + "\n4\n",
			expectedContent: "A\nB\nC\n",
		},
		{
			cmd:             []string{"1,5w " + path + "\n"},
			buffer:          dummyFile,
			init:            position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:          position{start: 1, end: 5, dot: len(dummyFile), addrc: 2},
			expectedOutput:  "10\n",
			expectedContent: "A\nB\nC\nD\nE\n",
		},
		{cmd: []string{"w \n"}, err: ErrNoFileName},
		{cmd: []string{"Wno\n"}, err: ErrUnexpectedCmdSuffix},
	}
	for _, tt := range tests {
		os.Remove(path)
		t.Run(strings.Join(tt.cmd, "\n"), func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdout(&b), WithStderr(&b))
			)
			ted.printErrors = true
			if tt.err == nil {
				setupMemoryFile(ted, tt.buffer)
				setPosition(ted, tt.init)
			}
			for _, cmd := range tt.cmd {
				ted.in = strings.NewReader(cmd)
				ted.tokenizer = nil
				if err := ted.Do(); err != tt.err {
					t.Errorf("expected error %q, got %q", tt.err, err)
				}
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(tt.expect, got) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
			if b.String() != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
			// Don't verify the file if we expect an error
			if tt.err == nil {
				f, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				buf, err := io.ReadAll(f)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.Close(); err != nil {
					t.Fatal(err)
				}
				if string(buf) != tt.expectedContent {
					t.Fatalf("expected file content to be %q, got %q", tt.expectedContent, string(buf))
				}
			}
		})
	}
}

func TestCmdEncrypt(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	setupMemoryFile(ted, dummyFile)
	ted.printErrors = true
	tests := []struct {
		cmd string
		err error
	}{
		{cmd: "1x\n", err: ErrUnexpectedAddress},
		{cmd: "x#\n", err: ErrInvalidCmdSuffix},
		{cmd: "x\n", err: ErrCryptUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
		})
	}
}

func TestCmdLineNumber(t *testing.T) {
	var (
		b    bytes.Buffer
		ted  = New(WithStdout(&b), WithStderr(&b))
		last = len(dummyFile)
	)
	setupMemoryFile(ted, dummyFile)
	ted.printErrors = true
	tests := []struct {
		cmd    string
		init   position
		output string
		err    error
	}{
		{
			cmd:    "=\n",
			init:   position{start: last, end: last, dot: last},
			output: fmt.Sprint(last) + "\n",
		},
		{
			cmd:    "=l\n",
			init:   position{start: last, end: last, dot: last},
			output: fmt.Sprint(last) + "\n" + dummyFile[last-1] + "$\n",
		},
		{
			cmd:    "=p\n",
			init:   position{start: last, end: last, dot: last, addrc: 2},
			output: fmt.Sprint(last) + "\n" + dummyFile[last-1] + "\n",
		},
		{
			cmd:    "2,3=\n",
			init:   position{start: last, end: last, dot: last},
			output: "3\n",
		},
		{cmd: "==\n", err: ErrInvalidCmdSuffix},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
		})
	}
}

func TestCmdExecute(t *testing.T) {
	var bufferName = "test"
	tests := []struct {
		cmd    []string
		output string
		path   string
		err    error
	}{
		{
			cmd:    []string{"! ls command.go\n"},
			output: "command.go\n!\n",
		},
		{
			cmd:    []string{"! ls command.go\n", "!!\n"},
			output: "command.go\n!\ncommand.go\n!\n",
		},
		{
			cmd:    []string{"! echo %\n"},
			path:   bufferName,
			output: bufferName + "\n!\n",
		},
		{cmd: []string{"15!\n"}, err: ErrUnexpectedAddress},
		{cmd: []string{"!\n"}, err: ErrNoCmd},
		{cmd: []string{"!!\n"}, err: ErrNoPreviousCmd},
		{cmd: []string{"! echo %\n"}, err: ErrNoFileName},
	}
	for _, tt := range tests {
		var (
			b   bytes.Buffer
			ted = New(WithStdout(&b), WithStderr(&b))
		)
		setupMemoryFile(ted, dummyFile)
		ted.path = tt.path
		ted.printErrors = true
		t.Run(strings.Join(tt.cmd, "\n"), func(t *testing.T) {
			for _, cmd := range tt.cmd {
				ted.in = strings.NewReader(cmd)
				ted.tokenizer = nil
				if err := ted.Do(); err != tt.err {
					t.Fatalf("expected error %q, got %q", tt.err, err)
				}
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
		})
	}
}

func TestCmdNone(t *testing.T) {
	tests := []struct {
		cmd    string
		output string
		err    error
	}{
		{
			cmd:    "10\n",
			output: dummyFile[9] + "\n",
		},
		{cmd: "\n", err: ErrInvalidAddress},
		{cmd: "999\n", err: ErrInvalidAddress},
	}
	for _, tt := range tests {
		var (
			b   bytes.Buffer
			ted = New(WithStdout(&b), WithStderr(&b))
		)
		ted.printErrors = true
		setupMemoryFile(ted, dummyFile)
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = nil
			if err := ted.Do(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			if b.String() != tt.output {
				t.Fatalf("expected output %q, got %q", tt.output, b.String())
			}
		})
	}
}

func TestCmdUnknown(t *testing.T) {
	tests := []string{"A\n", "B\n", "C\n"}
	for _, tt := range tests {
		var (
			b      bytes.Buffer
			ted    = New(WithStdout(&b), WithStderr(&b), WithSilent(false))
			expect = ErrUnknownCmd
		)
		ted.printErrors = true
		setupMemoryFile(ted, dummyFile)
		t.Run(tt, func(t *testing.T) {
			ted.in = strings.NewReader(tt)
			ted.tokenizer = nil
			if err := ted.Do(); err != expect {
				t.Fatalf("expected error %q, got %q", expect, err)
			}
		})
	}
}
