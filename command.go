package main

import (
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
	cmdSuffixPrint  cmdSuffix = 1 << iota // p - print the last line
	cmdSuffixList                         // l - list the last line
	cmdSuffixNumber                       // n - enumerate the last line
)

// subSuffix is a bitmask for the substitute command.
type subSuffix uint8

const (
	subGlobal    subSuffix = 1 << iota // g    complement previous global substiute suffix
	subNth                             // 0..9 repeat last substitution
	subPrint                           // p    complement previous print suffix
	subLastRegex                       // r    use last regex instead of last pattern
	subRepeat                          // \n   repeat last substitution
)

// getCmdSuffix extracts a command suffix which modifies the behaviour
// of the original command.
func (ed *Editor) getCmdSuffix() error {
	var done bool
	for !done {
		switch ed.tok {
		case 'p':
			ed.cs |= cmdSuffixPrint
			ed.token()
		case 'l':
			ed.cs |= cmdSuffixList
			ed.token()
		case 'n':
			ed.cs |= cmdSuffixNumber
			ed.token()
		default:
			done = true
		}
	}
	if ed.tok != '\n' && ed.tok != EOF {
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

// buildList builds a list of line indices for the lines that are marked
// by the the global command and the regular expression.
func (ed *Editor) buildList(r rune) error {
	var (
		delim  = ed.tok
		search string
		g      = (r == 'g' || r == 'G')
	)
	if delim == ' ' || delim == '\n' || delim == EOF {
		return ErrInvalidPatternDelim
	}
	ed.token()
	search = ed.scanStringUntil(delim)
	re, err := regexp.Compile(search)
	if err != nil {
		return err
	}
	if search == "" {
		if ed.re == nil {
			return ErrNoPrevPattern
		}
		re = ed.re
	}
	if r == 'G' || r == 'V' {
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
	}
	ed.list = []int{}
	for i := ed.start - 1; i < ed.end; i++ {
		if re.MatchString(ed.lines[i]) == g {
			ed.list = append(ed.list, i+1)
		}
	}
	return nil
}

func (ed *Editor) getCmdList() (string, error) {
	var (
		s, ln string
		done  bool
	)
	const sep = "\\"
	for {
		done = true
		ln = ed.scanString()
		if strings.HasSuffix(ln, sep) {
			ln = strings.TrimSuffix(ln, sep)
			done = false
			ed.token()
		}
		s += ln
		if !done {
			if ed.tok == EOF {
				return "", ErrUnexpectedEOF
			}
			continue
		}
		break
	}
	return s, nil
}

// global executes a command sequence globally. The start and end
// positions are set to the dot.
func (ed *Editor) global(r rune) error {
	var (
		interact = (r == 'G' || r == 'V')
		cmdlist  string
		err      error
		t        tokenizer
		ln       string
	)
	if !interact {
		cmdlist, err = ed.getCmdList()
		if err != nil {
			return err
		}
		if cmdlist == "" || cmdlist == "\n" {
			cmdlist = "p"
		}
	}
	defer func() {
		ed.g = false
		ed.tokenizer = &t
		ed.undohist = append(ed.undohist, ed.globalUndo)
		ed.globalCmd = cmdlist
	}()
	if interact && ed.tok == '\n' {
		ed.token()
	}
	var size int = len(ed.lines)
	for _, i := range ed.list {
		t = *ed.tokenizer
		ed.dot = i - (size - len(ed.lines))
		if interact {
			if err := ed.displayLines(ed.dot, ed.dot, ed.cs); err != nil {
				return err
			}
			ln, err = ed.getCmdList()
			if err != nil {
				return err
			}
			if ln == "" {
				continue
			} else if ln == "&" || ln == "&\n" {
				if ed.globalCmd == "" {
					return ErrNoPreviousCmd
				}
				ln = ed.globalCmd
			}
			t = *ed.tokenizer
			cmdlist = ln
		}
		ed.tokenizer = newTokenizer(strings.NewReader(cmdlist))
		ed.token()
		if err := ed.parse(); err != nil {
			return err
		}
		if err := ed.do(); err != nil {
			return err
		}
		if ed.cs > 0 {
			if err := ed.displayLines(ed.dot, ed.dot, ed.cs); err != nil {
				return err
			}
		}
		ed.tokenizer = &t
	}
	ed.cs = 0
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
			ed.dot = start
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
	if !ed.silent {
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
	if start >= 1 {
		start--
	}
	for i := start; i != end; i++ {
		var line string = ed.lines[i] + "\n"
		n, err := file.WriteString(line)
		if err != nil || n != len(line) {
			return ErrCannotWriteFile
		}
		siz += len(line)
	}
	ed.modified = false
	if !ed.silent {
		fmt.Fprintln(ed.err, siz)
	}
	return err
}

func (ed *Editor) substitute(re *regexp.Regexp, replace string, nth int, action *[]undoAction) error {
	var subs int
	for i := 0; i <= ed.end-ed.start; i++ {
		if !re.MatchString(ed.lines[i]) {
			continue
		}
		var (
			submatch = re.FindAllStringSubmatch(ed.lines[i], -1)
			matches  = re.FindAllStringIndex(ed.lines[i], nth)
		)
		for mi, match := range matches {
			if nth > 1 && mi != nth-1 {
				continue
			}
			var (
				start = match[0]
				end   = match[1]
				t     = newTokenizer(strings.NewReader(replace))
			)
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
			var replaced = ed.lines[i][:start] + r + ed.lines[i][end:]
			*action = append(*action,
				undoAction{typ: undoAdd,
					start: i + 1,
					end:   i + 1,
					dot:   ed.dot,
					lines: []string{ed.lines[i]},
				})
			*action = append(*action, undoAction{
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
	ed.re = re
	ed.replacestr = replace
	if subs == 0 && !ed.g {
		return ErrNoMatch
	} else if ed.g && ed.cs&cmdSuffixPrint|cmdSuffixNumber|cmdSuffixList > 0 {
		return ed.displayLines(ed.dot, ed.dot, ed.cs)
	}
	return nil
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
		if addr := ed.dot + 1; addr > len(ed.lines) {
			ed.dot = 1
		} else {
			ed.dot++
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
		} else if err := ed.buildList(mod); err != nil {
			return err
		}
		ed.g = true
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
		if ed.error != nil {
			return explainError{err: ed.error}
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
		var err = ed.displayLines(ed.start, ed.end, ed.cs)
		ed.cs = 0
		return err
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
		path = strings.TrimPrefix(path, " ")
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		return ed.readFile(path)
	case 's':
		var (
			nth   int = 1
			err   error
			delim rune
		)
		ed.token()
		for {
			switch {
			case ed.tok == '\n':
				ed.ss |= subRepeat
			case ed.tok == 'g':
				ed.ss |= subGlobal
				ed.token()
			case ed.tok == 'p':
				ed.ss |= subPrint
				ed.token()
			case ed.tok == 'r':
				// TODO(thimc): substitute: Implement 'r'
				ed.ss |= subLastRegex
				ed.token()
			case unicode.IsDigit(ed.tok):
				// TODO(thimc):  ubstitute: Implement '0'..'9'
				nth, err = ed.scanNumber()
				if err != nil {
					return ErrNumberOutOfRange
				}
				ed.ss |= subNth
				ed.ss &= ^subGlobal
			default:
				if ed.ss > 0 {
					return ErrInvalidCmdSuffix
				}
			}
			if ed.ss < 1 || ed.tok == '\n' || ed.tok == EOF {
				break
			}
		}
		if ed.ss > 0 && ed.re == nil {
			return ErrNoPrevPattern
		} else if ed.ss&subGlobal > 0 {
			nth = -1
		}
		delim = ed.tok
		ed.token()
		if delim == ' ' || delim == '\n' && ed.peek() == '\n' {
			return ErrInvalidPatternDelim
		}
		var (
			search  = ed.scanStringUntil(delim)
			replace = ed.scanStringUntil(delim)
			suffix  = ed.scanString()
			re      *regexp.Regexp
		)
		re, err = regexp.Compile(search)
		if err != nil {
			return err
		}
		if ed.ss&subRepeat > 0 {
			re = ed.re
			replace = ed.replacestr
		}
		if ed.ss&subPrint > 0 {
			ed.cs |= cmdSuffixPrint
		}
		if search == "" {
			if ed.re == nil {
				return ErrNoPrevPattern
			}
			re = ed.re
		}
		if replace == "%" {
			if ed.replacestr == "" {
				return ErrNoPreviousSub
			}
			replace = ed.replacestr
		}
		for _, ch := range suffix {
			switch {
			case ch == '\n' || ch == EOF:
				break
			case unicode.IsDigit(ch):
				nth, err = strconv.Atoi(string(ch))
				if err != nil {
					return ErrNumberOutOfRange
				}
			case ch == 'g':
				nth = -1
			case ch == 'p':
				ed.cs |= cmdSuffixPrint
			case ch == 'l':
				ed.cs |= cmdSuffixList
			case ch == 'n':
				ed.cs |= cmdSuffixNumber
			default:
				return ErrInvalidCmdSuffix
			}
		}
		if err := ed.check(ed.dot, ed.dot); err != nil {
			return err
		}
		return ed.substitute(re, replace, nth, &action)
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
			mod  = ed.tok
			quit = ed.token()
			path string
		)
		if !unicode.IsSpace(quit) && quit != 'Q' && quit != 'q' {
			return ErrUnexpectedCmdSuffix
		}
		ed.token()
		path = ed.scanString()
		if path == "" {
			if ed.path == "" {
				return ErrNoFileName
			}
			path = ed.path
		}
		if ed.addrc == 0 && len(ed.lines) < 1 {
			ed.start = 0
			ed.end = 0
		} else if err := ed.check(1, len(ed.lines)); err != nil {
			return err
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		if err := ed.writeFile(path, mod, ed.start, ed.end); err != nil {
			return err
		}
		if path[0] != '!' {
			ed.modified = false
		}
		if !ed.modified && quit == 'q' {
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
		if err := ed.check(ed.start, ed.dot+1); err != nil {
			return err
		} else if unicode.IsDigit(ed.tok) {
			var s string
			for unicode.IsDigit(ed.tok) {
				s += string(ed.tok)
				ed.token()
			}
			var err error
			ed.scroll, err = strconv.Atoi(s)
			if err != nil {
				return ErrNumberOutOfRange
			}
		}
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		var err = ed.displayLines(ed.end, min(len(ed.lines), ed.end+ed.scroll), ed.cs)
		ed.cs = 0
		return err
	case '=':
		ed.token()
		if err := ed.getCmdSuffix(); err != nil {
			return err
		}
		var n = ed.end
		if ed.addrc < 1 {
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
	return ErrUnknownCmd
}

// explainError is a type of error that is used to explain the editors actual error.
type explainError struct{ err error }

func (e explainError) Error() string { return e.err.Error() }
