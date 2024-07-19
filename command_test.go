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
		cmd, input     string
		buffer         []string
		init           position
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "1,3a",
			input:          "X\nY\nZ\n.\n",
			buffer:         []string{"A", "B", "C"},
			expectedBuffer: []string{"A", "B", "C", "X", "Y", "Z"},
			init:           position{start: 3, end: 3, dot: 3, addrc: 0},
			expect:         position{start: 1, end: 3, dot: 6, addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd + "\n" + tt.input)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
			}
		})
	}
}

func TestCmdChange(t *testing.T) {
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	tests := []struct {
		cmd, input     string
		buffer         []string
		expect         position
		expectedBuffer []string
	}{
		{
			cmd:            "1,3c",
			input:          "X\nY\nZ\n.\n",
			buffer:         []string{"A", "B", "C", "D", "E", "F"},
			expectedBuffer: []string{"X", "Y", "Z", "D", "E", "F"},
			expect:         position{start: 1, end: 3, dot: 3, addrc: 2},
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, tt.buffer)
			ted.in = strings.NewReader(tt.cmd + "\n" + tt.input)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
			}
		})
	}
}

func TestCmdEdit(t *testing.T) {
	var path = "dummy"
	if err := createDummyFile(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	tests := []struct {
		cmd            string
		expect         position
		expectedBuffer []string
		expectedOutput string
		err            error
	}{
		{
			cmd:            "e !ls ed.go\n",
			expect:         position{start: 0, end: 0, dot: 1, addrc: 0},
			expectedBuffer: []string{"ed.go"},
			expectedOutput: "6\n",
		},
		{
			cmd:            "e " + path + "\n",
			expect:         position{start: 0, end: 0, dot: len(dummyFile), addrc: 0},
			expectedBuffer: dummyFile,
			expectedOutput: "52\n",
		},
		{
			cmd:            "5e\n",
			expect:         position{start: 0, end: 0, dot: 0, addrc: 1},
			expectedBuffer: nil,
			err:            ErrUnexpectedAddress,
		},
		{
			cmd:            "e\n",
			expectedBuffer: nil,
			err:            ErrNoFileName,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			var (
				b   bytes.Buffer
				ted = New(WithStdin(strings.NewReader(tt.cmd)), WithStdout(&b), WithStderr(&b))
			)
			ted.printErrors = true
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
			}
			if b.String() != tt.expectedOutput {
				t.Fatalf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdFile(t *testing.T) {
	var path = "dummy"
	if err := createDummyFile(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	tests := []struct {
		cmd              string
		err              error
		expectedFilename string
		expectedOutput   string
	}{
		{
			cmd:              "f filename",
			expectedFilename: "filename",
			expectedOutput:   "filename\n",
		},
		{
			cmd: "f",
			err: ErrNoFileName,
		},
		{
			cmd: "2f",
			err: ErrUnexpectedAddress,
		},
		{
			cmd: "f !ls",
			err: ErrInvalidRedirection,
		},
		{
			cmd: "f!",
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
		ted    = New(WithStdout(io.Discard), WithStderr(io.Discard))
		buffer = dummyFile
		last   = len(buffer)
	)
	tests := []struct {
		cmd            string
		init           position
		expect         position
		expectedBuffer []string
		err            error
	}{
		{
			cmd:            "g	A	d\n",
			init:           position{start: last, end: last, dot: last},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 0},
			expectedBuffer: buffer[1:],
		},
		{
			cmd:            "v/A/d\n",
			init:           position{start: last, end: last, dot: last},
			expect:         position{start: 2, end: 2, dot: 2, addrc: 0},
			expectedBuffer: buffer[:1],
		},
		{
			cmd:            "v\n",
			init:           position{start: last, end: last, dot: last},
			expect:         position{start: 1, end: last, dot: last},
			expectedBuffer: buffer,
			err:            ErrInvalidPatternDelim,
		},
		{
			cmd:            "g a d \n",
			init:           position{start: last, end: last, dot: last},
			expect:         position{start: 1, end: last, dot: last},
			expectedBuffer: buffer,
			err:            ErrInvalidPatternDelim,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			setupMemoryFile(ted, buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected %q, got %q", tt.expectedBuffer, ted.Lines)
			}
		})
	}
}

func TestCmdHelp(t *testing.T) {
	var (
		ted    = New(WithStdin(strings.NewReader("2h\n")), WithStdout(io.Discard), WithStderr(io.Discard))
		expect = ErrNoFileName
	)
	ted.printErrors = true
	expect = ErrUnexpectedAddress
	if err := ted.Do(); err != expect {
		t.Fatalf("expected error %q, got %q", expect, err)
	}
	setupMemoryFile(ted, dummyFile)
	ted.in = strings.NewReader("2h\n")
	if err := ted.Do(); err != expect {
		t.Fatalf("expected error %q, got %q", expect, err)
	}
	ted.in = strings.NewReader("h\n")
	expect = ErrUnexpectedAddress
	if err := ted.Do(); err != expect {
		t.Fatalf("expected error %q, got %q", expect, err)
	}
}

func TestCmdHelpToggle(t *testing.T) {
	var (
		ted    = New(WithStdin(strings.NewReader("2H\n")), WithStdout(io.Discard), WithStderr(io.Discard))
		expect = ErrNoFileName
	)
	expect = ErrDefault
	if err := ted.Do(); err != expect {
		t.Fatalf("expected error %q, got %q", expect, err)
	}

	ted.in = strings.NewReader("H\n")
	expect = ErrUnexpectedAddress
	if err := ted.Do(); err != expect {
		t.Fatalf("expected error %q, got %q", expect, err)
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
			cmd:            "1,3i",
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
			ted.in = strings.NewReader(tt.cmd + "\n" + tt.input)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
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
		buffer         []string
		init           position
		expect         position
		expectedBuffer string
	}{
		{
			cmd:            "1,5m9\n",
			buffer:         dummyFile,
			init:           position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:         position{start: 1, end: 5, dot: 9, addrc: 1},
			expectedBuffer: "F\nG\nH\nI\nA\nB\nC\nD\nE\nJ\nK\nL\nM\nN\nO\nP\nQ\nR\nS\nT\nU\nV\nW\nX\nY\nZ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd + "\n")
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
			expect := strings.Split(tt.expectedBuffer, "\n")
			if !reflect.DeepEqual(ted.Lines, expect) {
				t.Fatalf("expected buffer %q, got %q", expect, ted.Lines)
			}
		})
	}
}

func TestCmdRead(t *testing.T) {
	var (
		b    bytes.Buffer
		ted  = New(WithStdout(&b), WithStderr(&b))
		path = "dummy_read"
	)
	ted.printErrors = true
	if err := createDummyFile(path); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	ted.in = strings.NewReader("r")
	if err := ted.Do(); err != ErrUnexpectedCmdSuffix {
		t.Fatalf("expected %q, got %q", ErrUnexpectedCmdSuffix, err)
	}
	ted.in = strings.NewReader("r\n")
	if err := ted.Do(); err != ErrNoFileName {
		t.Fatalf("expected %q, got %q", ErrNoFileName, err)
	}
	b.Reset()
	ted.in = strings.NewReader("r " + path + "\n")
	if err := ted.Do(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(dummyFile, ted.Lines) {
		t.Fatalf("expected buffer: %q, got %q", dummyFile, ted.Lines)
	}
}

func TestCmdSubstitute(t *testing.T) {
	var (
		b   bytes.Buffer
		buf = []string{
			"A A A A A",
			"A A A A A",
			"B B B B B",
			"B B B B B",
			"C C C C C",
			"C C C C C",
			"D D D D D",
			"D D D D D",
		}
		last = len(buf)
	)
	tests := []struct {
		cmd            string
		expect         position
		expectedOutput string
		expectedBuffer []string
		err            error
	}{
		{
			cmd:            ",s/A/X/gp\n",
			expectedBuffer: []string{"X X X X X", "X X X X X", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: last, dot: 2, addrc: 2},
			expectedOutput: "X X X X X\n",
		},
		{
			cmd:            ",s/A/X/g\n",
			expectedBuffer: []string{"X X X X X", "X X X X X", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: last, dot: 2, addrc: 2},
		},
		{
			cmd:            "1s/A/X/\n",
			expectedBuffer: []string{"X A A A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
		},
		{
			cmd:            "1s/A/X/g\n",
			expectedBuffer: []string{"X X X X X", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
		},
		{
			cmd:            "1s/A/X/3\n",
			expectedBuffer: []string{"A A X A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
		},
		{
			cmd:            "1s/A/&X/3\n",
			expectedBuffer: []string{"A A AX A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"},
			expect:         position{start: 1, end: 1, dot: 1, addrc: 1},
		},

		{
			cmd:            ",s/X/Y/g\n",
			expectedBuffer: buf,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            ErrNoMatch,
		},
		{
			cmd:            ",s//Y/\n",
			expectedBuffer: buf,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            ErrNoPrevPattern,
		},
		{
			cmd:            ",s/X/%/\n",
			expectedBuffer: buf,
			expect:         position{start: 1, end: last, dot: last, addrc: 2},
			err:            ErrNoPreviousSub,
		},
	}
	for _, tt := range tests {
		t.Run(string(tt.cmd), func(t *testing.T) {
			b.Reset()
			var ted = New(WithStdout(&b), WithStderr(io.Discard))
			setupMemoryFile(ted, buf)
			ted.in = strings.NewReader(tt.cmd)
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
			if !reflect.DeepEqual(tt.expectedBuffer, ted.Lines) {
				t.Fatalf("expected buffer: %q, got %q", tt.expectedBuffer, ted.Lines)
			}
			if b.String() != tt.expectedOutput {
				t.Fatalf("expected output: %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdScroll(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	tests := []struct {
		cmd            string
		buffer         []string
		init           position
		expect         position
		expectedOutput string
	}{
		{
			cmd:            "2z6\n",
			buffer:         dummyFile,
			init:           position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:         position{start: 1, end: 2, dot: 8, addrc: 1},
			expectedOutput: strings.Join(dummyFile[1:8], "\n") + "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
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
			if b.String() != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}

func TestCmdTransfer(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	tests := []struct {
		cmd            string
		buffer         []string
		init           position
		expect         position
		expectedBuffer string
	}{
		{
			cmd:            "1,5t3\n",
			buffer:         dummyFile,
			init:           position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:         position{start: 1, end: 5, dot: 8, addrc: 1},
			expectedBuffer: "A\nB\nC\nA\nB\nC\nD\nE\nD\nE\nF\nG\nH\nI\nJ\nK\nL\nM\nN\nO\nP\nQ\nR\nS\nT\nU\nV\nW\nX\nY\nZ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
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
			expect := strings.Split(tt.expectedBuffer, "\n")
			if !reflect.DeepEqual(ted.Lines, expect) {
				t.Fatalf("expected buffer %q, got %q", expect, ted.Lines)
			}
		})
	}
}

func TestCmdWrite(t *testing.T) {
	var (
		b    bytes.Buffer
		ted  = New(WithStdout(&b), WithStderr(&b))
		path = "dummy_write"
	)
	os.Remove(path)
	defer os.Remove(path)
	tests := []struct {
		cmd             string
		buffer          []string
		init            position
		expect          position
		expectedOutput  string
		expectedContent string
	}{
		{
			cmd:             "1,5w " + path + "\n",
			buffer:          dummyFile,
			init:            position{start: len(dummyFile), end: len(dummyFile), dot: len(dummyFile)},
			expect:          position{start: 1, end: 5, dot: len(dummyFile), addrc: 2},
			expectedOutput:  "10\n",
			expectedContent: "A\nB\nC\nD\nE\n",
		},
	}
	for _, tt := range tests {
		os.Remove(path)
		t.Run(tt.cmd, func(t *testing.T) {
			b.Reset()
			setupMemoryFile(ted, tt.buffer)
			setPosition(ted, tt.init)
			ted.in = strings.NewReader(tt.cmd)
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
			if b.String() != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
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
		})
	}

}

func TestCmdEncrypt(t *testing.T) {
	var (
		b   bytes.Buffer
		ted = New(WithStdout(&b), WithStderr(&b))
	)
	ted.printErrors = true
	ted.in = strings.NewReader("1x\n")
	if err := ted.Do(); err != ErrUnexpectedAddress {
		t.Fatalf("expected error %q, got %q", ErrUnexpectedAddress, err)
	}
	ted.in = strings.NewReader("x\n")
	if err := ted.Do(); err != ErrCryptUnavailable {
		t.Fatalf("expected error %q, got %q", ErrCryptUnavailable, err)
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
			cmds:   []string{"2,3c\ntest\n.", "u\n"},
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 2, end: 2, dot: last},
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
				if err := ted.Do(); err != nil {
					t.Errorf("expected no error, got %q", err)
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
			if !reflect.DeepEqual(ted.Lines, dummyFile) {
				t.Errorf("expected buffer %q, got %q", dummyFile, ted.Lines)
			}
			if b.String() != tt.expectedOutput {
				t.Errorf("expected output %q, got %q", tt.expectedOutput, b.String())
			}
		})
	}
}
