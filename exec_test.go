package main

import (
	"bytes"
	"fmt"
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
	defaultErr := fmt.Sprintf("%s\n", ErrDefault.Error())
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
		{cmd: "h", cur: cursor{first: lc, second: lc, dot: lc}, output: ""},
		{cmd: "H", cur: cursor{first: lc, second: lc, dot: lc}, output: "", keep: true},

		// i - insert
		{cmd: "ip\nworld\n.", cur: cursor{first: lc, second: lc, dot: lc}, output: "world\n"},
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}},
		{cmd: "ip\nhi\n.", cur: cursor{dot: 1}, keep: true, output: "hi\n", buf: []string{"hi"}},

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
		{cmd: "q", cur: cursor{first: 1, second: 2, dot: 1, addrc: 2}},
		{cmd: "1,2d", cur: cursor{first: 1, second: 2, dot: 1, addrc: 2}},
		{cmd: "Q", cur: cursor{first: 1, second: 2, dot: 1, addrc: 2}, keep: true},

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
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}},
		{cmd: "w", output: "0\n", keep: true},
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
		{cmd: "!echo \\%", cur: cursor{first: lc, second: lc, dot: lc}, output: "%\n!\n"},
		{cmd: "!echo %", cur: cursor{first: lc, second: lc, dot: lc}, output: fmt.Sprintf("%s\n!\n", dummy.path)},

		// ================================================================

		// a - append
		{cmd: "az", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// c - change
		{cmd: "cz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: ",d", cur: cursor{first: 1, second: lc, dot: 0, addrc: 2}, err: nil},
		{cmd: "c", keep: true, err: ErrInvalidAddress, output: defaultErr},

		// d - delete
		{cmd: "d", keep: true, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "dz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: fmt.Sprintf("1,%dd", lc+1), cur: cursor{first: 1, second: 1, dot: lc, addrc: 2}, err: ErrInvalidAddress, output: defaultErr},

		// e - open file
		{cmd: "1e", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "ez", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix, output: defaultErr},
		{cmd: "1d", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}},
		{cmd: "e", cur: cursor{first: 1, second: 1, dot: 1}, err: ErrFileModified, keep: true, output: defaultErr},
		{cmd: "e -non-existing-file-name-", cur: cursor{first: lc, second: lc}, err: ErrCannotReadFile, output: defaultErr},

		// f - filename
		{cmd: "fz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix, output: defaultErr},
		{cmd: "1f", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "f !", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidRedirection, output: defaultErr},
		{cmd: "f", cur: cursor{first: lc, second: lc, dot: lc}, path: true, err: ErrNoFileName, output: defaultErr},

		// v / V / g / G - global
		{cmd: ",d", cur: cursor{first: 1, second: lc, dot: 0, addrc: 2}},
		{cmd: "g/./p", cur: cursor{first: 1, second: 0, dot: 0}, keep: true, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "g/.*/g/.*/p", cur: cursor{first: 1, second: 1, dot: 1}, err: ErrCannotNestGlobal, output: defaultErr},
		{cmd: "2,5g A p", cur: cursor{first: 2, second: 5, dot: lc, addrc: 2}, err: ErrInvalidPatternDelim, output: defaultErr},
		{cmd: "g/A/\\", cur: cursor{first: 1, second: lc, dot: lc}, err: ErrUnexpectedEOF, output: defaultErr},
		{cmd: "G/A.*/\n&", cur: cursor{first: 1, second: lc, dot: 1}, output: "A\n" + defaultErr, err: ErrNoPreviousCmd},
		{cmd: "G/.*/\n,d", cur: cursor{first: 1, second: lc, dot: -24, addrc: 2}, output: "A\n" + defaultErr, err: ErrInvalidAddress},
		{cmd: "G/.*/\n\\", cur: cursor{first: 1, second: lc, dot: 1}, output: "A\n" + defaultErr, err: ErrUnexpectedEOF},
		{cmd: "G/.*", cur: cursor{first: 1, second: lc, dot: 1}, output: "A\n" + defaultErr, err: ErrUnexpectedEOF},
		{cmd: "G/.*\n\n", cur: cursor{first: 1, second: lc, dot: 2}, output: "A\nB\n" + defaultErr, err: ErrUnexpectedEOF},
		{cmd: "g/\n", cur: cursor{first: 1, second: lc, dot: lc}, err: ErrNoPrevPattern, output: defaultErr},
		{cmd: "g/(abc/", cur: cursor{first: 1, second: lc, dot: lc}, err: &syntax.Error{Code: syntax.ErrorCode("missing closing )"), Expr: "(abc"}, output: defaultErr},
		{cmd: "Gz", cur: cursor{first: 1, second: lc, dot: lc}, err: ErrNoPrevPattern, output: defaultErr},

		// h / H - error message
		{cmd: "1h", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "1H", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "Hz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: "1x", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnknownCmd, output: defaultErr},
		{cmd: "h", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnknownCmd, output: ErrUnknownCmd.Error() + "\n", keep: true},

		// TODO: Fix output
		// {cmd: "dz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		// {cmd: "h", cur: cursor{first: lc, second: lc, dot: lc}, keep: true, output: ErrInvalidCmdSuffix.Error()},

		// i - insert
		{cmd: "iz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// j - join
		{cmd: "1,2jz", cur: cursor{first: 1, second: 2, dot: lc, addrc: 2}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: "4,2j", cur: cursor{first: 4, second: 2, dot: lc, addrc: 2}, err: ErrInvalidAddress, output: defaultErr},

		// k - mark
		{cmd: "k!", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidMark, output: defaultErr},
		{cmd: "k!z", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}, buf: []string{}},
		{cmd: "ka", keep: true, err: ErrInvalidAddress, output: defaultErr},

		// l, n, p - print
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}},
		{cmd: "p", keep: true, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "pz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}, buf: []string{}},
		{cmd: "p", keep: true, err: ErrInvalidAddress, output: defaultErr},

		// m - move
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}},
		{cmd: "m5", keep: true, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "m1z", cur: cursor{first: lc, second: lc, dot: lc, addrc: 1}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: "1,5mz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrDestinationExpected, output: defaultErr},
		{cmd: "m", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrDestinationExpected, output: defaultErr},
		{cmd: "1,5m2", cur: cursor{first: 1, second: 5, dot: lc, addrc: 1}, err: ErrInvalidDestination, output: defaultErr},
		//{cmd: "1,2m1", cur: cursor{first: 1, second: 2, dot: lc, addrc: 2}, err: ErrInvalidDestination, output: defaultErr},
		//{cmd: "1m1a", cur: cursor{first: 1, second: 2, dot: 0, addrc: 1}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// P - prompt
		{cmd: "1P", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "Pq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// q / q - quit
		{cmd: "1d", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}, err: nil},
		{cmd: "q", cur: cursor{first: 1, second: 1, dot: 1}, keep: true, err: ErrFileModified, output: defaultErr},
		{cmd: "qq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: "1Q", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "Qq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// r - read
		{cmd: "rq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix, output: defaultErr},
		{cmd: "r non-existing-file", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrCannotReadFile, output: defaultErr},
		{cmd: "r", cur: cursor{first: lc, second: lc, dot: lc}, path: true, err: ErrNoFileName, output: defaultErr},
		{cmd: "r !non-existing-binary", cur: cursor{first: lc, second: lc, dot: lc}, err: &exec.Error{}, output: defaultErr},
		{cmd: "r !", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrNoCmd, output: defaultErr},
		// TODO(thimc): unsuccesful r (read) test with a command suffix. NOTE: I don't even know how to test this.

		// s - substitute
		{cmd: "spz", cur: cursor{first: slc, second: slc, dot: slc}, err: ErrInvalidCmdSuffix, sub: true, output: defaultErr},
		{cmd: ",s", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrNoPrevPattern, sub: true, output: defaultErr},
		{cmd: ",s/A/B/q", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrInvalidCmdSuffix, sub: true, output: defaultErr},
		{cmd: ",s/X/Y/", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrNoMatch, sub: true, output: defaultErr},
		{cmd: ",s//Y/", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, err: ErrNoPrevPattern, sub: true, output: defaultErr},
		{cmd: "s/(abc/", cur: cursor{first: lc, second: lc, dot: lc, addrc: 0}, err: &syntax.Error{Code: syntax.ErrorCode("missing closing )"), Expr: "(abc"}, output: defaultErr},
		{cmd: ",s/A/%/p", cur: cursor{first: 1, second: slc, dot: slc, addrc: 2}, sub: true, err: ErrNoPreviousSub, output: defaultErr},

		// t - transfer
		{cmd: fmt.Sprintf("%dt5", lc+2), cur: cursor{first: lc, second: lc, dot: lc, addrc: 1}, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "1,5tz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrDestinationExpected, output: defaultErr},
		{cmd: "1,5t5z", cur: cursor{first: 1, second: 5, dot: lc, addrc: 1}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: ",d", cur: cursor{first: 1, second: lc, addrc: 2}, buf: []string{}},
		{cmd: "1t2", cur: cursor{addrc: 1}, keep: true, err: ErrInvalidAddress, output: defaultErr},
		{cmd: ",d", cur: cursor{first: 1, second: lc, dot: 0, addrc: 2}, err: nil},
		{cmd: "1,5t2", cur: cursor{addrc: 1}, err: ErrInvalidAddress, keep: true, output: defaultErr},

		{cmd: "u", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrNothingToUndo, output: defaultErr},
		{cmd: "uq", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},
		{cmd: "1u", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},

		// w - write
		{cmd: "wz", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnexpectedCmdSuffix, output: defaultErr},
		{cmd: "w", cur: cursor{first: slc, second: slc, dot: slc}, sub: true, err: ErrNoFileName, output: defaultErr},
		{cmd: "wq", cur: cursor{first: slc, second: slc, dot: slc}, sub: true, err: ErrNoFileName, output: defaultErr},
		{cmd: "Wq", cur: cursor{first: slc, second: slc, dot: slc}, sub: true, err: ErrNoFileName, output: defaultErr},
		{cmd: "W", cur: cursor{first: slc, second: slc, dot: slc}, sub: true, err: ErrNoFileName, output: defaultErr},
		{cmd: "99,100w", cur: cursor{first: lc, second: lc, dot: lc, addrc: 1}, output: defaultErr, err: ErrInvalidAddress},
		{cmd: "w /root/no-access", cur: cursor{first: 1, second: lc, dot: lc}, output: defaultErr, err: ErrCannotOpenFile},
		{cmd: "1d", cur: cursor{first: 1, second: 1, dot: 1, addrc: 1}},
		{cmd: "Wq", cur: cursor{first: 1, second: lc - 1, dot: 1}, sub: true, err: ErrFileModified, keep: true, output: "50\n" + defaultErr},
		{cmd: fmt.Sprintf("WQ %s", tmp.Name()), cur: cursor{first: 1, second: lc, dot: lc}, output: "52\n"},

		// z - scroll
		{cmd: "1z1234567891234567891234567890", cur: cursor{first: 1, second: 1, dot: lc, addrc: 1}, err: ErrNumberOutOfRange, output: defaultErr},
		{cmd: "z", cur: cursor{first: 1, second: lc + 1, dot: lc}, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "5zq", cur: cursor{first: 1, second: 5, dot: lc, addrc: 1}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// = - line count
		{cmd: "=q", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrInvalidCmdSuffix, output: defaultErr},

		// ! - shell escape
		{cmd: "!", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrNoCmd, output: defaultErr},
		{cmd: "5!", cur: cursor{first: 5, second: 5, dot: lc, addrc: 1}, err: ErrUnexpectedAddress, output: defaultErr},
		{cmd: "!nonexistingcommnad", cur: cursor{first: lc, second: lc, dot: lc}, err: &exec.ExitError{}, output: defaultErr},
		{cmd: "!echo %", cur: cursor{first: lc, second: lc, dot: lc}, path: true, err: ErrNoFileName, output: defaultErr},

		// no/unknown command
		{cmd: "\n", cur: cursor{first: 1, second: lc + 1, dot: lc}, err: ErrInvalidAddress, output: defaultErr},
		{cmd: "@", cur: cursor{first: lc, second: lc, dot: lc}, err: ErrUnknownCmd, output: defaultErr},
	}

	var ed *Editor
	var output bytes.Buffer
	for _, test := range tests {
		t.Run(test.cmd, func(t *testing.T) {
			output.Reset()
			buf := dummy
			if test.sub {
				buf = subBuffer
			}
			if !test.keep {
				copy(dummy.lines, dlines)
				copy(subBuffer.lines, slines)
				ed = NewEditor(
					WithStdout(&output),
					WithStderr(&output),
					withBuffer(buf),
				)
			}
			WithStdin(strings.NewReader(test.cmd))(ed)
			if test.path {
				ed.file.path = ""
			}
			defer func() { recover() }() // allow tests that call os.Exit()
			err := ed.run()
			if err != test.err {
				if xerr, ok := err.(*exec.ExitError); ok {
					if _, ok := test.err.(*exec.ExitError); ok {
						// TODO(thimc): compare the actual exit error
						_ = xerr
					}
				} else if synerr, ok := err.(*syntax.Error); ok {
					// TODO(thimc): verify the regexp.syntax.Error
					_ = synerr
				} else {
					t.Fatalf("want %q, got %q", test.err, err)
				}
			}
			if err != nil {
				ed.errorln(ed.verbose, err)
			}
			if output.String() != test.output {
				t.Fatalf("want stdout/stderr %q, got %q", test.output, output.String())
			}
			if test.buf != nil && strings.Join(test.buf, "\n") != strings.Join(ed.file.lines, "\n") {
				t.Fatalf("want buffer\n%+q\ngot buffer\n%+q", test.buf, ed.file.lines)
			}
			if ed.cursor != test.cur {
				t.Fatalf("want %+v, got %+v", test.cur, ed.cursor)
			}
		})
	}
}
