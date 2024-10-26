package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func setPosition(ted *Editor, pos position) {
	ted.start = pos.start
	ted.end = pos.end
	ted.dot = pos.dot
	ted.addrc = pos.addrc
}

var dummyFile = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
}

// setupMemoryFile initializes a in-memory buffer
func setupMemoryFile(ed *Editor, buf []string) {
	ed.lines = make([]string, len(buf))
	copy(ed.lines, buf)
	ed.path = "test"
	ed.dot = len(buf)
	ed.printErrors = true
}

// createDummyFile creates a dummy file `fname` containing `dummyFile`.
func createDummyFile(fname string, t *testing.T) {
	file, err := os.Create(fname)
	if err != nil {
		t.Fatalf("create dummy file: %q", err)
	}
	defer file.Close()
	if _, err = file.WriteString(strings.Join(dummyFile, "\n")); err != nil {
		t.Fatalf("write dummy file: %q", err)
	}
}

func TestParser(t *testing.T) {
	var last = len(dummyFile)
	tests := []struct {
		cmd    string
		init   position
		expect position
		empty  bool
		err    error
	}{
		{
			cmd:    "",
			init:   position{start: 1, end: 1, dot: last},
			expect: position{start: last, end: last, dot: last},
		},
		{
			cmd:    "	8",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 8, end: 8, dot: last, addrc: 1},
		},
		{
			cmd:    fmt.Sprint(last),
			init:   position{start: last, end: last, dot: last},
			expect: position{start: last, end: last, dot: last, addrc: 1},
		},
		{
			cmd:    "1,5",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 1, end: 5, dot: last, addrc: 2},
		},
		{
			cmd:    "+",
			init:   position{start: 2, end: 2, dot: 2},
			expect: position{start: 3, end: 3, dot: 2, addrc: 1},
		},
		{
			cmd:    "-",
			init:   position{start: 3, end: 3, dot: 3},
			expect: position{start: 2, end: 2, dot: 3, addrc: 1},
		},
		{
			cmd:    "^",
			init:   position{start: 3, end: 3, dot: 3},
			expect: position{start: 2, end: 2, dot: 3, addrc: 1},
		},
		{
			cmd:    ".,+5",
			init:   position{start: 4, end: 4, dot: 4},
			expect: position{start: 4, end: 9, dot: 4, addrc: 2},
		},
		{
			cmd:    "-2,+5",
			init:   position{start: 5, end: 5, dot: 5},
			expect: position{start: 3, end: 10, dot: 5, addrc: 2},
		},
		{
			cmd:    ",",
			init:   position{start: 5, end: 5, dot: 5},
			expect: position{start: 1, end: last, dot: 5, addrc: 2},
		},
		{
			cmd:    "10,",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 10, end: 10, dot: last, addrc: 1},
		},
		{
			cmd:    "6,%",
			init:   position{start: 5, end: 5, dot: 5},
			expect: position{start: 1, end: last, dot: 5, addrc: 2},
		},
		{
			cmd:    "3,;",
			init:   position{start: 5, end: 5, dot: 5},
			expect: position{start: 3, end: 26, dot: 3, addrc: 2},
		},
		{
			cmd:    ";",
			init:   position{start: 5, end: 5, dot: 5},
			expect: position{start: 5, end: last, dot: 5, addrc: 2},
		},
		{
			cmd:    "/D/\n//",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 4, end: 4, dot: last, addrc: 1},
		},
		{
			cmd:    "?E?\n??",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 5, end: 5, dot: last, addrc: 1},
		},
		{
			cmd:    "'a",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 1, end: 1, dot: last, addrc: 1},
		},
		{
			cmd:    "1,/^F$/-",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 1, end: 5, dot: last, addrc: 2},
		},
		{
			cmd:    "?D?,?B?+",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 4, end: 3, dot: last, addrc: 2},
		},
		{
			cmd:    "/^A$/,/^D$/+",
			init:   position{start: last, end: last, dot: last},
			expect: position{start: 1, end: 5, dot: last, addrc: 2},
		},

		// Error cases, some of these positions that are expected do not make
		// sense but since we are not executing any command _after_ them, i.e.
		// we are not checking the range with [ed.check], so it is fine.
		{cmd: "?test?", empty: true, err: ErrNoMatch},
		{cmd: "'z", err: ErrInvalidMark},
		{cmd: "'f", err: ErrInvalidAddress},
		{cmd: "1'a", err: ErrInvalidAddress},
		{cmd: "1," + fmt.Sprint(last+20), err: ErrInvalidAddress, expect: position{start: 0, end: 1, dot: 0, addrc: 2}},
		{cmd: "']", err: ErrInvalidMark},
		{cmd: "1.", err: ErrInvalidAddress},
		{cmd: "-999", err: ErrInvalidAddress, expect: position{addrc: 1}},
		{cmd: "//", err: ErrNoPrevPattern},
		{cmd: "1//", err: ErrInvalidAddress},
		{cmd: "/non_existing_text/", err: ErrNoMatch},
		{cmd: "??", err: ErrNoPrevPattern},
		{cmd: fmt.Sprint(last + 1), err: ErrInvalidAddress, expect: position{addrc: 1}},
	}
	for _, tt := range tests {
		var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
		if !tt.empty {
			setupMemoryFile(ted, dummyFile)
			ted.mark[0] = 1 // Set the mark a as the first line to test it later.
		}
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			setPosition(ted, tt.init)
			ted.tokenizer = newTokenizer(ted.in)
			ted.consume()
			if err := ted.parse(); err != tt.err {
				t.Fatalf("expected error %q, got %q", tt.err, err)
			}
			got := position{start: ted.start, end: ted.end, dot: ted.dot, addrc: ted.addrc}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
		})
	}

	tests2 := []struct {
		cmd    string
		expect position
	}{
		{cmd: "/F/\n", expect: position{start: 6, end: 6, dot: 52, addrc: 1}},
		{cmd: "//\n", expect: position{start: 32, end: 32, dot: 6, addrc: 1}},
		{cmd: "//\n", expect: position{start: 6, end: 6, dot: 32, addrc: 1}},
		{cmd: "?D?\n", expect: position{start: 4, end: 4, dot: 6, addrc: 1}},
		{cmd: "??\n", expect: position{start: 30, end: 30, dot: 4, addrc: 1}},
		{cmd: "??\n", expect: position{start: 4, end: 4, dot: 30, addrc: 1}},
	}
	var ted = New(WithStdout(io.Discard), WithStderr(io.Discard))
	setupMemoryFile(ted, append(dummyFile, dummyFile...))
	for _, tt := range tests2 {
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			ted.tokenizer = newTokenizer(ted.in)
			ted.consume()
			if err := ted.parse(); err != nil {
				t.Fatalf("parse: %q", err)
			}
			got := position{start: ted.start, end: ted.end, dot: ted.dot, addrc: ted.addrc}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Fatalf("expected %+v, got %+v", tt.expect, got)
			}
			// TODO(thimc): To avoid executing Do() in the Parser tests we explicitly
			// set the dot to the start address which is usually done in do() in the
			// case where we don't have a follow up command, i.e the token being '\n'.
			ted.dot = ted.start
		})
	}

}
