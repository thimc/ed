package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// cmdSuffix is a bitmask for most commands which modifies the way the
// command handles the last line. It can print, list or enumerate it.
type cmdSuffix uint8

const (
	cmdSuffixPrint  cmdSuffix = 1 << iota // p
	cmdSuffixList                         // l
	cmdSuffixNumber                       // n
)

// subSuffix is a bitmask for the substitute command.
type subSuffix uint8

const (
	subGlobal    subSuffix = 1 << iota // g
	subNth                             // 0...9
	subPrint                           // p
	subLastRegex                       // r
)

// getCmdSuffix extracts a command suffix which modifies the behaviour
// of the original command.
func (ed *Editor) getCmdSuffix() error {
	for {
		var found bool
	loop:
		switch ed.tok {
		case 'p':
			ed.cs |= cmdSuffixPrint
			ed.token()
			found = true
			goto loop
		case 'l':
			ed.cs |= cmdSuffixList
			ed.token()
			found = true
			goto loop
		case 'n':
			ed.cs |= cmdSuffixNumber
			ed.token()
			found = true
			goto loop
		default:
			found = true
		}
		if found {
			break
		}
	}
	if ed.tok != '\n' {
		return ErrInvalidCmdSuffix
	}
	return nil
}

// getThirdAddress extracts the third legal address used by the `move`
// and `transfer` commands.
func (ed *Editor) getThirdAddress() (int, error) {
	var start, end = ed.start, ed.end
	if err := ed.parse(); err != nil {
		return -1, err
	} else if ed.addrc == 0 {
		return -1, ErrDestinationExpected
	} else if ed.end < 0 || ed.end > len(ed.lines) {
		return -1, ErrInvalidAddress
	}
	var addr = ed.end
	ed.start = start
	ed.end = end
	return addr, nil
}

// undo undoes the last command and restores the current address
// to what it was before the last command.
func (ed *Editor) undo() (err error) {
	// TODO(thimc): Undo 'u' is its own inverse so it should push an
	// inverse action of whatever the action is.
	if len(ed.undohist) < 1 {
		return ErrNothingToUndo
	}
	var operation = ed.undohist[len(ed.undohist)-1]
	ed.undohist = ed.undohist[:len(ed.undohist)-1]
	for n := len(operation) - 1; n >= 0; n-- {
		var (
			op     = operation[n]
			before = ed.lines[:op.start-1]
			after  = ed.lines[op.start-1:]
		)
		switch op.typ {
		case undoDelete:
			after = ed.lines[op.end:]
			ed.lines = append(before, after...)
		case undoAdd:
			ed.lines = append(before, append(op.lines, after...)...)
		}
		ed.dot = op.dot
		ed.modified = true
	}
	return nil
}

// global executes a command sequence globally. The start and end
// positions are set to the dot.
func (ed *Editor) global(r rune) error {
	var (
		interactive = (r == 'G' || r == 'V')
		invert      = (r == 'v' || r == 'V')
		tokenizer   = ed.tokenizer
		search, cmd string
	)
	// TODO(thimc): The V and G commands take a suffix
	// according to the spec.  Doing this now will result
	// in "invalid command suffix" error.
	// if interactive {
	// 	if err := ed.getCmdSuffix(); err != nil {
	// 		return err
	// 	}
	// }
	var delim = ed.tok
	ed.token()
	if delim == ' ' || delim == '\n' {
		return ErrInvalidPatternDelim
	}
	var s, e = ed.start, ed.end
	search = ed.scanStringUntil(delim)
	if ed.tok != EOF && ed.tok != '\n' {
		cmd = ed.scanString()
		if strings.HasSuffix(cmd, string(delim)) {
			cmd = cmd[:len(cmd)-1]
		}
	}
	defer func() {
		ed.g = false
		ed.tokenizer = tokenizer
		ed.undohist = append(ed.undohist, ed.globalUndo)
		ed.globalCmd = cmd
	}()
	for idx := s - 1; idx <= e; idx++ {
		if idx >= len(ed.lines) {
			continue
		}
		matched, err := regexp.MatchString(search, ed.lines[idx])
		if err != nil {
			return err
		}
		if (!invert && !matched) || (invert && matched) {
			continue
		}
		if cmd == "" && !interactive {
			cmd = "p"
		}
		if interactive {
			if err := ed.displayLines(ed.dot, ed.dot, ed.cs); err != nil {
				return err
			}
			r := bufio.NewReader(ed.in)
		input:
			for {
				select {
				case <-ed.sigintch:
					return ed.interrupt()
				default:
					ln, err := r.ReadString('\n')
					if err != nil {
						return err
					}
					if len(ln) > 1 {
						ln = ln[:len(ln)-1]
					}
					cmd = ln
					if cmd == "&" {
						if ed.globalCmd == "" {
							return ErrNoPreviousCmd
						}
						cmd = ed.globalCmd
					}
					break input
				}
			}
		}
		for _, c := range strings.Split(cmd, "\\") {
			ed.dot = idx + 1
			ed.start = ed.dot
			ed.end = ed.dot
			ed.tokenizer = newTokenizer(strings.NewReader(c + "\n"))
			ed.token()
			ed.g = true
			if err = ed.do(); err != nil {
				return err
			}
			if e >= len(ed.lines) {
				e = len(ed.lines) - 1
				idx--
			}
		}

	}
	ed.start = ed.dot
	ed.end = ed.dot
	return nil
}

// deleteLines deletes the addressed lines from the buffer. The dot is
// set to the line after the range if one exists, otherwise it is set to
// the line before the range.
func (ed *Editor) deleteLines(start, end int, action *[]undoAction) error {
	undolines := make([]string, end-start+1)
	copy(undolines, ed.lines[start-1:end])
	*action = append(*action, undoAction{
		typ:   undoAdd,
		start: start,
		end:   start,
		dot:   ed.dot,
		lines: undolines,
	})
	ed.lines = append(ed.lines[:start-1], ed.lines[end:]...)
	ed.dot = start - 1
	ed.start = min(start, len(ed.lines))
	ed.modified = true
	return nil
}

func (ed *Editor) appendLines(start int, action *[]undoAction) error {
loop:
	for {
		select {
		case <-ed.sigintch:
			return ed.interrupt()
		default:
			ed.dot = start
			line, err := ed.tokenizer.ReadString('\n')
			if err != nil {
				break loop
			}
			if len(line) > 1 {
				line = line[:len(line)-1]
			}
			if line == "." {
				break loop
			}
			if len(ed.lines) == start {
				ed.lines = append(ed.lines, line)
			} else {
				ed.lines = append(ed.lines[:start], append([]string{line}, ed.lines[start:]...)...)
			}
			start++
			*action = append(*action, undoAction{
				typ:   undoDelete,
				start: start,
				end:   start,
				dot:   ed.dot,
				lines: ed.lines[start-1 : start],
			})
			ed.modified = true
		}
	}
	return nil
}

func (ed *Editor) joinLines(start, end int, action *[]undoAction) error {
	var undolines = make([]string, end-start+1)
	copy(undolines, ed.lines[start-1:end])
	var lines = strings.Join(ed.lines[start-1:end], "")
	*action = append(*action, undoAction{
		typ:   undoAdd,
		start: start,
		end:   start + len(undolines) - 1,
		dot:   ed.dot,
		lines: undolines,
	})
	var joined []string = append(append([]string{}, ed.lines[:start-1]...), lines)
	ed.lines = append(joined, ed.lines[end:]...)
	*action = append(*action, undoAction{
		typ:   undoDelete,
		start: start,
		end:   start,
		dot:   ed.dot,
		lines: []string{lines},
	})
	ed.dot = start
	ed.modified = true
	return nil
}

func (ed *Editor) displayLines(start, end int, flags cmdSuffix) error {
	if start == 0 {
		return ErrInvalidAddress
	}
	for start--; start != end; {
		ed.dot = start + 1
		var ln string
		if flags&cmdSuffixNumber != 0 {
			ln = fmt.Sprintf("%d\t", ed.dot)
		}
		if flags&cmdSuffixList != 0 {
			var q = strings.Replace(strconv.QuoteToASCII(ed.lines[start]), "$", "\\$", -1)
			ln += fmt.Sprintf("%s$", q[1:len(q)-1])
		} else {
			ln += ed.lines[start]
		}
		fmt.Fprintln(ed.out, ln)
		start++
	}
	return nil
}

func (ed *Editor) moveLines(addr int, action *[]undoAction) error {
	var lines = make([]string, ed.end-ed.start+1)
	copy(lines, ed.lines[ed.start-1:ed.end])
	*action = append(*action, undoAction{
		typ:   undoAdd,
		start: ed.start,
		end:   ed.start + len(lines) - 1,
		dot:   ed.dot,
		lines: lines,
	})
	ed.lines = append(ed.lines[:ed.start-1], ed.lines[ed.end:]...)
	if addr-len(lines) < 0 {
		addr = len(lines) + addr
	}
	ed.lines = append(ed.lines[:addr-len(lines)], append(lines, ed.lines[addr-len(lines):]...)...)

	var undolines = make([]string, len(lines))
	copy(undolines, ed.lines[addr-len(lines):addr])
	*action = append(*action, undoAction{
		typ:   undoDelete,
		start: addr - ed.start + 1,
		end:   addr - ed.start + len(undolines),
		dot:   addr,
		lines: undolines,
	})

	ed.dot = addr
	if addr < ed.start {
		ed.dot += ed.end - ed.start + 1
	}
	return nil
}

func (ed *Editor) copyLines(addr int, action *[]undoAction) error {
	var lines = make([]string, ed.end-ed.start+1)
	copy(lines, ed.lines[ed.start-1:ed.end])
	ed.lines = append(ed.lines[:addr], append(lines, ed.lines[addr:]...)...)
	ed.end = addr + len(lines)
	*action = append(*action, undoAction{
		typ:   undoDelete,
		start: addr + 1,
		end:   addr + len(lines),
		dot:   ed.dot,
		lines: lines,
	})
	ed.end = len(lines)
	ed.dot = addr + len(lines)
	ed.modified = true
	return nil
}

func (ed *Editor) readFile(path string) error {
	if path == "" || path == "\n" {
		if ed.path == "" {
			return ErrNoFileName
		}
		path = ed.path
	} else if unicode.IsSpace(rune(path[0])) {
		return ErrInvalidFileName
	}
	var (
		siz int64
		cmd bool = path[0] == '!'
	)
	if cmd {
		path = path[1:]
		lines, err := ed.shell(path)
		if err != nil {
			return err
		}
		ed.lines = lines
		siz = int64(len(strings.Join(lines, " "))) + 1
	} else {
		f, err := os.Open(path)
		if err != nil {
			return ErrCannotOpenFile
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			return ErrCannotReadFile
		}
		s, err := os.Stat(path)
		if err != nil {
			return ErrCannotReadFile
		}
		siz = s.Size() + 1
		if err := f.Close(); err != nil {
			return ErrCannotCloseFile
		}
		var lines = strings.Split(strings.TrimRight(string(buf), "\n"), "\n")
		ed.lines = lines
		ed.path = path
	}
	if !ed.scripted {
		fmt.Fprintln(ed.err, siz)
	}
	ed.dot = len(ed.lines)
	return nil
}

func (ed *Editor) writeFile(path string, mod rune, start, end int) error {
	var (
		file *os.File
		err  error
	)
	if mod == 'W' {
		file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	} else {
		file, err = os.Create(path)
	}
	if err != nil {
		return err
	}
	defer file.Close()
	var siz int
	for i := start - 1; i < end; i++ {
		var line string = ed.lines[i]
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
		siz += len(line) + 1
	}
	ed.modified = false
	if !ed.scripted {
		fmt.Fprintln(ed.err, siz)
	}
	return err
}

// do executes a command on a range defined by start and end.
func (ed *Editor) do() (err error) {
	var action = make([]undoAction, 0)
	defer func() {
		if len(action) > 0 {
			if !ed.g {
				ed.undohist = append(ed.undohist, action)
			} else {
				ed.globalUndo = append(ed.globalUndo, action...)
			}
		}
	}()
	ed.cs = 0
	ed.ss = 0
	ed.skipWhitespace()
	switch ed.tok {
	case 'a':
		ed.token()
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.appendLines(ed.end, &action)
	case 'c':
		ed.token()
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if err := ed.deleteLines(ed.start, ed.end, &action); err != nil {
			return err
		}
		return ed.appendLines(ed.dot, &action)
	case 'd':
		ed.token()
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if err := ed.deleteLines(ed.start, ed.end, &action); err != nil {
			return err
		}
		ed.dot += 1
		if addr := ed.dot; addr > len(ed.lines) {
			ed.dot = addr
		}
		return nil
	case 'E', 'e':
		var mod = ed.tok
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if ed.modified && mod != 'E' {
			ed.modified = false
			return ErrFileModified
		}
		if !unicode.IsSpace(ed.tok) {
			return ErrUnexpectedCmdSuffix
		}
		ed.skipWhitespace()
		ed.globalUndo = nil
		ed.undohist = nil
		ed.lines = nil
		action = nil
		ed.modified = false
		// TODO(thce): Edit/edit should take a command suffix.
		// Doing so will also result in an error currently.
		// if err := ed.getCmdSuffix(); err != nil {
		// 	return err
		// }
		return ed.readFile(ed.scanString())
	case 'f':
		ed.token()
		var path string
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		} else if !unicode.IsSpace(ed.tok) {
			return ErrUnexpectedCmdSuffix
		}
		if path = ed.scanString(); path == "" {
			if ed.path == "" {
				return ErrNoFileName
			}
			path = ed.path
		}

		if path[0] == ' ' {
			path = path[1:]
		}
		if path[0] == '!' {
			return ErrInvalidRedirection
		}
		ed.path = path
		fmt.Fprintln(ed.out, ed.path)
		return nil
	case 'V', 'G', 'v', 'g':
		var mod = ed.tok
		ed.token()
		if ed.g {
			return ErrCannotNestGlobal
		} else if err := ed.check(1, len(ed.lines)); err != nil {
			return err
		}
		return ed.global(mod)
	case 'h', 'H':
		var mod = ed.tok
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if mod == 'H' {
			ed.printErrors = !ed.printErrors
		}
		return ed.error
	case 'i':
		ed.token()
		if ed.end == 0 {
			ed.end = 1
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.appendLines(ed.end-1, &action)
	case 'j':
		ed.token()
		if err := ed.check(ed.dot, ed.dot+1); err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if ed.start != ed.end {
			if err := ed.joinLines(ed.start, ed.end, &action); err != nil {
				return err
			}
		}
		return nil
	case 'k':
		ed.token()
		if ed.end == 0 {
			return ErrInvalidAddress
		}
		var m = ed.tok
		ed.token()
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if !unicode.IsLower(m) {
			return ErrInvalidMark
		}
		ed.mark[m-'a'] = ed.end
		return nil
	case 'l', 'n', 'p':
		switch ed.tok {
		case 'l':
			ed.cs |= cmdSuffixList
		case 'n':
			ed.cs |= cmdSuffixNumber
		case 'p':
			ed.cs |= cmdSuffixPrint
		}
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.displayLines(ed.start, ed.end, ed.cs)
	case 'm':
		ed.token()
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		addr, err := ed.getThirdAddress()
		if err != nil {
			return err
		}
		if ed.start <= addr && addr < ed.end {
			return ErrInvalidDestination
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.moveLines(addr, &action)
	case 'P':
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		ed.showPrompt = !ed.showPrompt
		return nil
	case 'q', 'Q':
		var mod = ed.tok
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if mod == 'q' && ed.modified {
			ed.modified = false
			return ErrFileModified
		}
		os.Exit(0)
	case 'r':
		if !unicode.IsSpace(ed.token()) {
			return ErrUnexpectedCmdSuffix
		} else if ed.addrc == 0 {
			ed.end = len(ed.lines)
		}
		var path string
		if path = ed.scanString(); path == "" {
			if ed.path == "" {
				return ErrNoFileName
			}
		}
		if path[0] == ' ' {
			path = path[1:]
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.readFile(path)
	case 's':
		ed.token()
		var (
			delim = ed.tok
			nth   = 1
			re    *regexp.Regexp
		)
		ed.token()
		var (
			search  = ed.scanStringUntil(delim)
			replace = ed.scanStringUntil(delim)
		)
		if ed.tok != '\n' {
			for ed.tok != EOF && ed.tok != '\n' {
				switch {
				case ed.tok == 'g':
					ed.ss |= subGlobal
					nth = -1
					ed.token()
				case ed.tok == 'r':
					ed.ss |= subLastRegex
					// TODO(thimc): implement 'r' for the substitute command:
					// (.,.)s
					//  Repeats the last substitution.  This form of the s command accepts
					//  a count suffix n, or any combination of the characters r, g, and p.
					//  The r suffix causes the regular expression of the last search to be
					//  used instead of that of the last substitution.
					ed.token()
				case unicode.IsDigit(ed.tok):
					ed.ss |= subNth
					nth, err = strconv.Atoi(string(ed.tok))
					if err != nil {
						return err
					}
					ed.token()
				case ed.tok == 'p':
					ed.cs |= cmdSuffixPrint
					ed.token()
				case ed.tok == 'l':
					ed.cs |= cmdSuffixList
					ed.token()
				case ed.tok == 'n':
					ed.cs |= cmdSuffixNumber
					ed.token()
				default:
					return ErrInvalidCmdSuffix
				}
			}
		}
		if search == "" {
			if ed.search == "" {
				return ErrNoPrevPattern
			}
			search = ed.search
		}
		if replace == "%" {
			if ed.replacestr == "" {
				return ErrNoPreviousSub
			}
			replace = ed.replacestr
		}
		re, err = regexp.Compile(search)
		if err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		var subs int
		for i := 0; i <= ed.end-ed.start; i++ {
			if !re.MatchString(ed.lines[i]) {
				continue
			}
			var (
				submatch = re.FindAllStringSubmatch(ed.lines[i], -1)
				matches  = re.FindAllStringIndex(ed.lines[i], nth)
			)
			for mn, match := range matches {
				if nth > 1 && mn != nth-1 {
					continue
				}
				var (
					start = match[0]
					end   = match[1]
				)
				var t = newTokenizer(strings.NewReader(replace))
				t.token()
				var r string
				for t.tok != EOF {
					if (t.tok != '\\' && t.peek() == '&') || (t.tokpos == 1 && t.tok == '&') {
						if t.tokpos > 1 {
							r += string(t.tok)
							t.token()
						}
						t.token()
						r += ed.lines[i][start:end]
						continue
					} else if t.tok == '\\' && unicode.IsDigit(t.peek()) {
						t.token()
						n, err := strconv.Atoi(string(t.tok))
						if err != nil {
							return ErrNumberOutOfRange
						}
						t.token()
						r += submatch[0][n-1]
						continue
					}
					r += string(t.tok)
					t.token()
				}
				var replaced = ed.lines[i][:start] + re.ReplaceAllString(search, r) + ed.lines[i][end:]
				action = append(action,
					undoAction{typ: undoAdd,
						start: i + 1,
						end:   i + 1,
						dot:   ed.dot,
						lines: []string{ed.lines[i]},
					})
				action = append(action, undoAction{
					typ:   undoDelete,
					start: i + 1,
					end:   i + 1,
					lines: []string{replaced},
				})
				ed.lines[i] = replaced
			}
			ed.dot = i + 1
			subs++
		}
		if subs == 0 && !ed.g {
			return ErrNoMatch
		} else if ed.cs&(cmdSuffixList|cmdSuffixNumber|cmdSuffixPrint) > 0 {
			return ed.displayLines(ed.dot, ed.dot, ed.cs)
		}
		return nil
	case 't':
		ed.token()
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		addr, err := ed.getThirdAddress()
		if err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.copyLines(addr, &action)
	case 'u':
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.undo()
	case 'W', 'w':
		var (
			t    = ed.tok
			mod  = ed.token()
			path string
		)
		if !unicode.IsSpace(ed.tok) {
			return ErrUnexpectedCmdSuffix
		} else if path = ed.scanString(); path == "" {
			if ed.path == "" {
				return ErrNoFileName
			}
		}
		if path[0] == ' ' {
			path = path[1:]
		}
		if ed.addrc == 0 && len(ed.lines) < 1 {
			ed.start = 0
			ed.end = 0
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if err := ed.writeFile(path, t, ed.start, ed.end); err != nil {
			return err
		} else if path[0] != '!' {
			ed.modified = false
			return nil
		} else if ed.modified && !ed.scripted && mod == 'q' {
			os.Exit(0)
		}
		return nil
	case 'x':
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ErrCryptUnavailable
	case 'z':
		ed.token()
		ed.start = 1
		var n int
		if err := ed.check(ed.start, ed.dot+1); err != nil {
			return err
		} else if unicode.IsDigit(ed.tok) {
			var s string
			for unicode.IsDigit(ed.tok) {
				s += string(ed.tok)
				ed.token()
			}
			var err error
			n, err = strconv.Atoi(s)
			if err != nil {
				return ErrNumberOutOfRange
			}
			// ed.token()
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.displayLines(ed.end, min(len(ed.lines), ed.end+n), ed.cs)
	case '=':
		ed.token()
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		var n = ed.end
		if ed.addrc > 1 {
			n = len(ed.lines)
		}
		fmt.Fprintln(ed.out, n)
		return nil
	case '!':
		ed.token()
		if ed.addrc > 0 {
			return ErrUnexpectedAddress
		}
		if ed.tok == EOF || ed.tok == '\n' {
			return ErrNoCmd
		}
		ed.skipWhitespace()
		var cmd = ed.scanString()
		ed.getCmdSuffix()
		output, err := ed.shell(cmd)
		if err != nil {
			return err
		}
		for i := range output {
			fmt.Fprintln(ed.out, output[i])
		}
		fmt.Fprintln(ed.out, "!")
		return nil
	case '\n':
		ed.start = 1
		if err := ed.check(ed.start, ed.dot+1); err != nil {
			return err
		}
		return ed.displayLines(ed.end, ed.end, 0)
	}
	ed.token()
	return ErrUnknownCmd
}
