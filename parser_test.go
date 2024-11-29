package main

import (
	"fmt"
	"io"
	"regexp/syntax"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	lc := len(dummy.lines)
	tests := []struct {
		cmd   string
		cur   cursor
		keep  bool
		empty bool
		perr  error // parsing error
		xerr  error // execution error
	}{
		{cmd: "2,4", cur: cursor{first: 2, second: 4, dot: lc, addrc: 2}},
		{cmd: "3;10", cur: cursor{first: 3, second: 10, dot: 3, addrc: 2}},
		{cmd: ";6", cur: cursor{first: lc, second: 6, dot: lc, addrc: 2}},
		{cmd: "3,6", cur: cursor{first: 3, second: 6, dot: lc, addrc: 2}},
		{cmd: "/C/,?G?", cur: cursor{first: 3, second: 7, dot: lc, addrc: 2}},
		{cmd: "/", cur: cursor{first: 7, second: 7, dot: 7, addrc: 1}, keep: true},
		{cmd: "1,?Z?", cur: cursor{first: 1, second: lc, dot: lc, addrc: 2}},
		{cmd: "1,", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}},
		{cmd: "5", cur: cursor{first: 5, second: 5, dot: lc, addrc: 1}},
		{cmd: ".,+7", cur: cursor{first: 5, second: 12, dot: 5, addrc: 2}, keep: true},
		{cmd: "'a", cur: cursor{first: 3, second: 3, dot: lc, addrc: 1}},
		{cmd: "%", cur: cursor{first: 1, second: lc, dot: lc, addrc: 2}},
		{cmd: "$", cur: cursor{first: lc, second: lc, dot: lc, addrc: 1}},
		{cmd: "-", cur: cursor{first: lc - 1, second: lc - 1, dot: lc, addrc: 1}},
		{cmd: "^", cur: cursor{first: lc - 1, second: lc - 1, dot: lc, addrc: 1}},
		{cmd: "+", cur: cursor{first: lc, second: lc, dot: lc - 1, addrc: 1}, keep: true},
		{cmd: "", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}},

		// error cases
		{cmd: "0", cur: cursor{first: 0, second: 0, dot: lc, addrc: 1}, xerr: ErrInvalidAddress},
		{cmd: fmt.Sprint(lc + 1), cur: cursor{first: lc, second: lc, dot: lc, addrc: 1}, perr: ErrInvalidAddress},
		{cmd: "2.5", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrInvalidAddress},
		{cmd: "//", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrNoPrevPattern},
		{cmd: "/next line", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrNoMatch},
		{cmd: "?prev line", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, empty: true, perr: ErrNoMatch},
		{cmd: "/(abc/", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: &syntax.Error{Code: syntax.ErrorCode("missing closing )"), Expr: "(abc"}},
		{cmd: "5,'h", cur: cursor{first: 5, second: 5, dot: lc, addrc: 1}, perr: ErrInvalidAddress},
		{cmd: "5/A/", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrInvalidAddress},
		{cmd: "5'A", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrInvalidAddress},
		{cmd: "'A", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrInvalidMark},
		{cmd: "'@", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrInvalidMark},
		{cmd: "9999999999999999999", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, perr: ErrNumberOutOfRange},
	}

	var ed *Editor
	for _, test := range tests {
		t.Run(fmt.Sprintf("%q", test.cmd), func(t *testing.T) {
			if !test.keep {
				ed = NewEditor(
					withBuffer(dummy),
					WithStdin(strings.NewReader(test.cmd)),
					WithStdout(io.Discard),
					WithStderr(io.Discard),
				)
			} else {
				_ = ed.exec() // Needed to validate the [cursor]
			}
			if test.empty {
				ed.file.lines = []string{}
			}
			ed.doInput(test.cmd)
			if err := ed.parse(); err != test.perr {
				if synerr, ok := err.(*syntax.Error); ok {
					// TODO: verify the regexp.syntax.Error
					_ = synerr
				} else {
					t.Fatalf("want parse error: %+v, got %+v", test.perr, err)
				}
			}
			if test.cur != ed.cursor {
				t.Fatalf("want %+v, got %+v", test.cur, ed.cursor)
			}
			if test.xerr != nil {
				if err := ed.exec(); err != test.xerr {
					t.Fatalf("want exec error: %+v, got %+v", test.xerr, err)
				}
			}
		})
	}
}
