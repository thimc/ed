package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp/syntax"
	"strings"
	"testing"
)

var (
	dummy = file{
		lines: []string{
			"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
			"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
		},
		mark: [25]int{3, 0},
		path: "#dummy",
	}
	subBuffer = file{
		lines: []string{
			"A A A A A",
			"A A A A A",
			"B B B B B",
			"B B B B B",
			"C C C C C",
			"C C C C C",
			"D D D D D",
			"D D D D D",
		},
	}
)

func withBuffer(f file) Option {
	return func(ed *Editor) { ed.file, ed.dot = f, len(f.lines) }
}

func TestEditor(t *testing.T) {
	dlines := make([]string, len(dummy.lines))
	copy(dlines, dummy.lines)
	slines := make([]string, len(subBuffer.lines))
	copy(slines, subBuffer.lines)

	tmp, err := os.CreateTemp(os.TempDir(), "ed")
	if err != nil {
		panic(err)
	}
	if _, err := tmp.WriteString(strings.Join(dummy.lines, "\n")); err != nil {
		panic(err)
	}
	if err := tmp.Close(); err != nil {
		panic(err)
	}
	defer os.Remove(tmp.Name())
	dummy.path = tmp.Name()

	lc := len(dummy.lines)
	slc := len(subBuffer.lines)
	tests := []struct {
		cmd    string
		cur    cursor
		output string
		err    error
		keep   bool // keeps the editor as it was in the previous test (rather than reinitializing it)
		sub    bool // use another buffer when testing substitutes
		path   bool // empty the file name for error testing
		buf    []string
	}{
		// a - append
		{cmd: "ap\nhello\nworld\n.", cur: cursor{first: lc, second: lc, dot: lc + 2}, output: "world\n", buf: append(dummy.lines, []string{"hello", "world"}...)},

		// c - change
		{cmd: "cp\nhello\nworld\n.", cur: cursor{first: lc, second: lc, dot: lc + 1}, output: "world\n"},

		// d - delete
		{cmd: "d", cur: cursor{first: lc, second: lc, dot: lc - 1}},

		// e / E - edit
		{cmd: fmt.Sprintf("e %s", tmp.Name()), cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc*2)},
		{cmd: fmt.Sprintf("E %s", tmp.Name()), cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc*2), keep: true},

		// f - file name
		{cmd: "f", cur: cursor{first: lc, second: lc, dot: lc}, output: dummy.path + "\n"},
		{cmd: "f test", cur: cursor{first: lc, second: lc, dot: lc}, output: "test\n"},

		// v / V / g / G - global
		{cmd: "g/.*/", cur: cursor{first: lc, second: lc, dot: lc}, output: strings.Join(dummy.lines, "\n") + "\n"},
		{cmd: "g/.*/p", cur: cursor{first: lc, second: lc, dot: lc}, output: strings.Join(dummy.lines, "\n") + "\n"},
		{cmd: "v/A/p", cur: cursor{first: lc, second: lc, dot: lc}, output: strings.Join(dummy.lines[1:], "\n") + "\n"},
		{cmd: "G/A.*/npl\np\np", cur: cursor{first: 2, second: 2, dot: 2}, output: "1\tA A A A A$\nA A A A A\n2\tA A A A A$\nA A A A A\n", sub: true},
		{cmd: "G/.*/\nn\n&\n&\n&\n&\n&\n&\n&\n", cur: cursor{first: slc, second: slc, dot: slc}, output: "A A A A A\n1\tA A A A A\nA A A A A\n2\tA A A A A\nB B B B B\n3\tB B B B B\nB B B B B\n4\tB B B B B\nC C C C C\n5\tC C C C C\nC C C C C\n6\tC C C C C\nD D D D D\n7\tD D D D D\nD D D D D\n8\tD D D D D\n", sub: true},
		{cmd: "V/A/\nn\n&\n&\n&\n&\n&\n&\n&\n", cur: cursor{first: slc, second: slc, dot: slc}, output: "B B B B B\n3\tB B B B B\nB B B B B\n4\tB B B B B\nC C C C C\n5\tC C C C C\nC C C C C\n6\tC C C C C\nD D D D D\n7\tD D D D D\nD D D D D\n8\tD D D D D\n", sub: true},

		// h / H - error message
		{cmd: "h", cur: cursor{first: lc, second: lc, dot: lc}},
		{cmd: "H", cur: cursor{first: lc, second: lc, dot: lc}},

		// i - insert
		{cmd: "ip\nworld\n.", cur: cursor{first: lc, second: lc, dot: lc}, output: "world\n"},
		// TODO(thimc): insert test when the file buffer is empty

		// j - join
		{cmd: fmt.Sprintf("%d,%dj", lc-1, lc), cur: cursor{first: lc - 1, second: lc, dot: lc, addrc: 2}},

		// m - move
		{cmd: "1,5m9", cur: cursor{first: 1, second: 5, dot: 9, addrc: 1}, buf: []string{"F", "G", "H", "I", "A", "B", "C", "D", "E", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}},
		{cmd: "m1", cur: cursor{first: lc, second: lc, dot: 2, addrc: 1}, buf: append([]string{dummy.lines[0]}, append([]string{dummy.lines[lc-1]}, dummy.lines[1:lc-1]...)...)},

		// k - mark
		{cmd: "kd", cur: cursor{first: lc, second: lc, dot: lc}},

		// p / n / l - print
		{cmd: ",p", cur: cursor{first: 1, second: lc, dot: lc, addrc: 2}, output: strings.Join(dummy.lines, "\n") + "\n"},
		{cmd: "n", cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\tZ\n", len(dummy.lines))},
		{cmd: "l", cur: cursor{first: lc, second: lc, dot: lc}, output: "Z$\n"},

		// P - prompt toggle
		{cmd: "P", cur: cursor{first: lc, second: lc, dot: lc}},

		// q - quit
		{cmd: "1,2d\nq", cur: cursor{first: 1, second: 2, dot: 1, addrc: 2}, keep: true},
		// TODO(thimc): succesful quit test

		// r - read
		{cmd: "r", cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc*2)},
		{cmd: fmt.Sprintf("r %s", tmp.Name()), cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc*2)},

		// s - substitute
		{cmd: ",s/A/X/gp", cur: cursor{first: 1, second: slc, dot: 2, addrc: 2}, output: "X X X X X\n", buf: []string{"X X X X X", "X X X X X", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"}, sub: true},
		{cmd: "1s/A/&X/3", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}, buf: []string{"A A AX A A", "A A A A A", "B B B B B", "B B B B B", "C C C C C", "C C C C C", "D D D D D", "D D D D D"}, sub: true},
		{cmd: `3,5s/ (.)(.)/_\2_\1X\2_/`, cur: cursor{first: 3, second: 5, dot: 5, addrc: 2}, buf: []string{"A A A A A", "A A A A A", "B_ _BX _B B B", "B_ _BX _B B B", "C_ _CX _C C C", "C C C C C", "D D D D D", "D D D D D"}, sub: true},
		{cmd: "s/.*/some/nl", cur: cursor{first: 8, second: 8, dot: 8}, output: "8\tsome$\n", sub: true},
		{cmd: "1s/A/TEST/", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}, sub: true},
		{cmd: "s", cur: cursor{first: 1, second: 1, dot: 1}, buf: append([]string{"TEST TEST A A A"}, subBuffer.lines[1:]...), keep: true, sub: true},
		{cmd: "s3", cur: cursor{first: 1, second: 1, dot: 1}, buf: append([]string{"TEST TEST A A TEST"}, subBuffer.lines[1:]...), keep: true, sub: true},
		{cmd: "sr", cur: cursor{first: 1, second: 1, dot: 1}, buf: append([]string{"TEST TEST TEST A TEST"}, subBuffer.lines[1:]...), keep: true, sub: true},
		{cmd: "sgp", cur: cursor{first: 1, second: 1, dot: 1}, buf: append([]string{"TEST TEST TEST TEST TEST"}, subBuffer.lines[1:]...), output: "TEST TEST TEST TEST TEST\n", keep: true, sub: true},
		{cmd: ",s/B.*/test", cur: cursor{first: 1, second: slc, dot: 4, addrc: 2}, output: "test\n", sub: true},
		{cmd: ",s/B.*/test/", cur: cursor{first: 1, second: slc, dot: 4, addrc: 2}, sub: true},

		{cmd: ",s/A/TEST/", cur: cursor{first: 1, second: slc, dot: 2, addrc: 2}, sub: true},
		{cmd: ",s/A/%/", cur: cursor{first: 1, second: slc, dot: 2, addrc: 2}, buf: append([]string{"TEST TEST A A A", "TEST TEST A A A"}, subBuffer.lines[2:]...), keep: true, sub: true},

		// t - transfer
		{cmd: "1,5t3", cur: cursor{first: 1, second: 5, dot: 8, addrc: 1}, output: ""},

		// u - undo

		// w / wq / W - write
		{cmd: "1w", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, output: "2\n"},
		{cmd: "w", cur: cursor{first: 1, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc*2)},
		{cmd: fmt.Sprintf("w %s", tmp.Name()), cur: cursor{first: 1, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc*2)},

		// z - scroll
		{cmd: "2z6", cur: cursor{first: 1, second: 2, dot: 8, addrc: 1}, output: strings.Join(dummy.lines[1:8], "\n") + "\n"},

		// = - line count
		{cmd: "=", cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\n", lc)},
		{cmd: "3,5=", cur: cursor{first: 3, second: 5, dot: lc, addrc: 2}, output: "5\n"},
		{cmd: "=p", cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%d\n%s\n", lc, dummy.lines[lc-1])},

		// ! - shell escape
		{cmd: "!echo hi", cur: cursor{first: lc, second: lc, dot: lc}, output: "hi\n!\n"},

		// ================================================================

		// a - append
		{cmd: "az", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},

		// c - change
		{cmd: "cz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},
		{cmd: ",d", cur: cursor{first: 1, second: lc, dot: 0, addrc: 2}, err: nil},
		{cmd: "c", keep: true, err: ErrInvalidAddress},

		// d - delete
		{cmd: "d", keep: true, err: ErrInvalidAddress},
		{cmd: "dz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},
		{cmd: fmt.Sprintf("1,%dd", lc+1), cur: cursor{first: 1, second: 1, dot: lc, addrc: 2}, err: ErrInvalidAddress},

		// e - open file
		{cmd: "1e", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "ez", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix},
		{cmd: "1d", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}},
		{cmd: "e", cur: cursor{first: 1, second: 1, dot: 1}, err: ErrFileModified, keep: true},
		{cmd: "e -non-existing-file-name-", cur: cursor{first: lc, second: lc}, err: ErrCannotReadFile},

		// f - filename
		{cmd: "fz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix},
		{cmd: "1f", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "f !", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidRedirection},
		{cmd: "f", cur: cursor{first: lc, second: lc, dot: lc}, path: true, err: ErrNoFileName},

		// v / V / g / G - global
		{cmd: ",d", cur: cursor{first: 1, second: lc, dot: 0, addrc: 2}},
		{cmd: "g/./p", cur: cursor{first: 1, second: 0, dot: 0}, keep: true, err: ErrInvalidAddress},
		{cmd: "g/.*/g/.*/p", cur: cursor{first: 1, second: 1, dot: 1}, err: ErrCannotNestGlobal},
		{cmd: "2,5g A p", cur: cursor{first: 2, second: 5, dot: lc, addrc: 2}, err: ErrInvalidPatternDelim},
		{cmd: "g/A/\\", cur: cursor{first: 1, second: lc, dot: lc}, err: ErrUnexpectedEOF},
		{cmd: "G/A.*/\n&", cur: cursor{first: 1, second: lc, dot: 1}, output: "A\n", err: ErrNoPreviousCmd},
		{cmd: "G/.*/\n,d", cur: cursor{first: 1, second: lc, dot: -24, addrc: 2}, output: "A\n", err: ErrInvalidAddress},
		{cmd: "G/.*/\n\\", cur: cursor{first: 1, second: lc, dot: 1}, output: "A\n", err: ErrUnexpectedEOF},
		{cmd: "G/.*", cur: cursor{first: 1, second: lc, dot: 1}, output: "A\n", err: ErrUnexpectedEOF},
		{cmd: "G/.*\n\n", cur: cursor{first: 1, second: lc, dot: 2}, output: "A\nB\n", err: ErrUnexpectedEOF},
		{cmd: "g/\n", cur: cursor{first: 1, second: lc, dot: lc}, err: ErrNoPrevPattern},
		{cmd: "g/(abc/", cur: cursor{first: 1, second: lc, dot: lc}, err: &syntax.Error{Code: syntax.ErrorCode("missing closing )"), Expr: "(abc"}},
		{cmd: "Gz", cur: cursor{first: 1, second: lc, dot: lc}, err: ErrNoPrevPattern},

		// H - toggle errors
		{cmd: "1h", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "1H", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "Hz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},

		// i - insert
		{cmd: "iz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},

		// j - join
		{cmd: "1,2jz", cur: cursor{first: 1, second: 2, dot: lc, addrc: 2}, err: ErrInvalidCmdSuffix},
		{cmd: "4,2j", cur: cursor{first: 4, second: 2, dot: lc, addrc: 2}, err: ErrInvalidAddress},

		// k - mark
		{cmd: "k!", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidMark},
		{cmd: "k!z", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},
		// TODO(thimc): k test when the file buffer is empty

		// l, n, p - print
		{cmd: "pz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},
		// TODO(thimc): print test when the file buffer is empty

		// m - move
		{cmd: "1,5mz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrDestinationExpected},
		{cmd: "m", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrDestinationExpected},
		{cmd: "1,5m2", cur: cursor{first: 1, second: 5, dot: lc, addrc: 1}, err: ErrInvalidDestination},
		//{cmd: "1,2m1", cur: cursor{first: 1, second: 2, dot: lc, addrc: 2}, err: ErrInvalidDestination},
		//{cmd: "1m1a", cur: cursor{first: 1, second: 2, dot: 0, addrc: 1}, err: ErrInvalidCmdSuffix},

		// P - prompt
		{cmd: "1P", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "Pq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},

		// q / q - quit
		{cmd: "1d", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}, err: nil},
		{cmd: "q", cur: cursor{first: 1, second: 1, dot: 1}, keep: true, err: ErrFileModified},
		{cmd: "qq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},
		{cmd: "1Q", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "Qq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},

		// r - read
		{cmd: "rq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix},
		{cmd: "r non-existing-file", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrCannotReadFile},
		{cmd: "r", cur: cursor{first: lc, second: lc, dot: lc}, path: true, err: ErrNoFileName},
		{cmd: "r !non-existing-binary", cur: cursor{first: lc, second: lc, dot: lc}, err: &exec.Error{}},
		{cmd: "r !", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrNoCmd},
		// TODO(thimc): unsuccesful r (read) test with a command suffix. NOTE: I don't even know how to test this.

		// s - substitute
		{cmd: "spz", cur: cursor{first: slc, second: slc, dot: slc}, err: ErrInvalidCmdSuffix, sub: true},
		{cmd: ",s", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrNoPrevPattern, sub: true},
		{cmd: ",s/A/B/q", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrInvalidCmdSuffix, sub: true},
		{cmd: ",s/X/Y/", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrNoMatch, sub: true},
		{cmd: ",s//Y/", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrNoPrevPattern, sub: true},
		{cmd: "s/(abc/", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, err: &syntax.Error{Code: syntax.ErrorCode("missing closing )"), Expr: "(abc"}},
		{cmd: ",s/A/%/p", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, sub: true, err: ErrNoPreviousSub},

		// t - transfer
		{cmd: fmt.Sprintf("%dt5", lc+2), cur: cursor{first: lc, second: lc, dot: lc, addrc: 1}, err: ErrInvalidAddress},
		{cmd: "1,5tz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrDestinationExpected},
		{cmd: "1,5t5z", cur: cursor{first: 1, second: 5, dot: lc, addrc: 1}, err: ErrInvalidCmdSuffix},

		// TODO(thimc): transfer test when the file buffer is empty
		{cmd: ",d", cur: cursor{first: 1, second: lc, dot: 0, addrc: 2}, err: nil},
		{cmd: "1,5t2", cur: cursor{addrc: 1}, err: ErrInvalidAddress, keep: true},

		// TODO(thimc): undo tests
		{cmd: "u", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrNothingToUndo},
		{cmd: "uq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},
		{cmd: "1u", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},

		// w - write
		{cmd: "wz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix},
		{cmd: "w", cur: cursor{first: slc, second: slc, dot: slc}, sub: true, err: ErrNoFileName},
		// TODO(thimc): wq tests

		// z - scroll
		{cmd: "1z1234567891234567891234567890", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrNumberOutOfRange},
		{cmd: "z", cur: cursor{first: 1, second: lc + 1, dot: lc}, err: ErrInvalidAddress},
		{cmd: "5zq", cur: cursor{first: 1, second: 5, dot: lc, addrc: 1}, err: ErrInvalidCmdSuffix},

		// = - line count
		{cmd: "=q", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix},

		// ! - shell escape
		{cmd: "!", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrNoCmd},
		{cmd: "5!", cur: cursor{first: 5, second: 5, dot: lc, addrc: 1}, err: ErrUnexpectedAddress},
		{cmd: "!nonexistingcommnad", cur: cursor{first: lc, second: lc, dot: lc}, err: &exec.ExitError{}},

		// no/unknown command
		{cmd: "\n", cur: cursor{first: 1, second: lc + 1, dot: lc}, err: ErrInvalidAddress},
		{cmd: "@", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnknownCmd},
	}

	var ed *Editor
	var stdout bytes.Buffer
	for _, test := range tests {
		t.Run(test.cmd, func(t *testing.T) {
			stdout.Reset()
			buf := dummy
			if test.sub {
				buf = subBuffer
			}
			if !test.keep {
				copy(dummy.lines, dlines)
				copy(subBuffer.lines, slines)
				ed = NewEditor(
					WithStdout(&stdout),
					WithStderr(io.Discard),
					withBuffer(buf),
				)
			}
			WithStdin(strings.NewReader(test.cmd))(ed)
			if test.path {
				ed.file.path = ""
			}
			if err := ed.run(); err != test.err {
				if xerr, ok := err.(*exec.ExitError); ok {
					if _, ok := test.err.(*exec.ExitError); ok {
						// TODO(thimc): compare the actual exit error
						_ = xerr
					}
				} else if synerr, ok := err.(*syntax.Error); ok {
					// TODO(thimc): verify the regexp.syntax.Error
					_ = synerr
				} else {
					t.Fatalf("want %+v, got %+v", test.err, err)
				}
			}
			if test.buf != nil && strings.Join(test.buf, "\n") != strings.Join(ed.file.lines, "\n") {
				t.Fatalf("want buffer\n%+q\ngot buffer\n%+q", test.buf, ed.file.lines)
			}
			if ed.cursor != test.cur {
				t.Fatalf("want %+v, got %+v", test.cur, ed.cursor)
			}
			if stdout.String() != test.output {
				t.Fatalf("want stdout %q, got %q", test.output, stdout.String())
			}
		})
	}
}
