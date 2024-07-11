package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"
	"unicode"
)

// doCommand executes a command on a range defined by start and end.
func (ed *Editor) doCommand() (err error) {
	var (
		undolines []string
		action    []undoAction
	)
	defer func() {
		if len(action) > 0 {
			if !ed.g {
				ed.undohist = append(ed.undohist, action)
			} else {
				ed.globalUndo = append(ed.globalUndo, action...)
			}
		}
	}()
	switch ed.tok {
	case 'a':
		return ed.cmdAppend(&action)
	case 'c':
		return ed.cmdChange(undolines, &action)
	case 'd':
		return ed.cmdDelete(undolines, &action)
	case 'E', 'e':
		return ed.cmdEdit(ed.tok == 'E')
	case 'f':
		return ed.cmdFilename()
	case 'V', 'G', 'v', 'g':
		return ed.cmdGlobal(ed.tok == 'G' || ed.tok == 'V', ed.tok == 'v' || ed.tok == 'V')
	case 'H':
		return ed.cmdToggleError()
	case 'h':
		return ed.cmdExplainError()
	case 'i':
		return ed.cmdInsert(&action)
	case 'j':
		return ed.cmdJoin(undolines, &action)
	case 'k':
		return ed.cmdMark()
	case 'm':
		return ed.cmdMove(undolines, &action)
	case 'l', 'n', 'p':
		return ed.cmdPrint()
	case 'P':
		return ed.cmdTogglePrompt()
	case 'q', 'Q':
		return ed.cmdQuit(ed.tok == 'Q')
	case 'r':
		return ed.cmdRead(&action)
	case 's':
		return ed.cmdSubstitute(undolines, &action)
	case 't':
		return ed.cmdTransfer(&action)
	case 'u':
		return ed.cmdUndo()
	case 'W', 'w':
		return ed.cmdWrite(ed.tok == 'W')
	case 'z':
		return ed.cmdScroll()
	case '=':
		return ed.cmdPrintLineNumber()
	case '!':
		return ed.cmdExecute()
	case 0, scanner.EOF:
		if ed.s.Pos().Offset == 0 {
			ed.dot++
			if ed.dot >= len(ed.Lines) {
				ed.dot = len(ed.Lines)
				err = ErrInvalidAddress
			}
			ed.start = ed.dot
			ed.end = ed.dot
			if err != nil {
				return err
			}
		}
		if ed.end-1 < 0 {
			return ErrInvalidAddress
		}
		fmt.Fprintln(ed.out, ed.Lines[ed.end-1])
		return err
	}
	return ErrUnknownCmd
}

func (ed *Editor) cmdAppend(action *[]undoAction) error {
	r := bufio.NewReader(ed.in)
loop:
	for {
		select {
		case <-ed.sigintch:
			fmt.Fprintln(ed.err, ErrDefault)
			break loop
		default:
			line, err := r.ReadString('\n')
			if err != nil {
				break loop
			}
			line = line[:len(line)-1]
			if line == "." {
				break loop
			}
			if len(ed.Lines) == ed.end {
				ed.Lines = append(ed.Lines, line)
			} else {
				ed.Lines = append(ed.Lines[:ed.end], append([]string{line}, ed.Lines[ed.end:]...)...)
			}
			ed.end++
			*action = append(*action, undoAction{typ: undoDelete, start: ed.end - 1, end: ed.end})
			ed.start = ed.end
			ed.dot = ed.end
			ed.dirty = true
		}
	}
	return nil
}

func (ed *Editor) cmdChange(undolines []string, action *[]undoAction) error {
	undolines = make([]string, ed.end-ed.start+1)
	copy(undolines, ed.Lines[ed.start-1:ed.end])
	*action = append(*action, undoAction{typ: undoAdd, start: ed.start, end: ed.start - 1, d: ed.start + len(undolines) - 1, lines: undolines})
	ed.Lines = append(ed.Lines[:ed.start-1], ed.Lines[ed.end:]...)

	ed.end = ed.start - 1
	r := bufio.NewReader(ed.in)
loop:
	for {
		select {
		case <-ed.sigintch:
			fmt.Fprintln(ed.err, ErrDefault)
			break loop
		default:
			line, err := r.ReadString('\n')
			if err != nil {
				break loop
			}
			line = line[:len(line)-1]
			if line == "." {
				break loop
			}
			if ed.end > len(ed.Lines) {
				ed.Lines = append(ed.Lines[:ed.end], line)
			} else {
				ed.Lines = append(ed.Lines[:ed.end], append([]string{line}, ed.Lines[ed.end:]...)...)
			}
			ed.end++
			*action = append(*action, undoAction{typ: undoDelete, start: ed.end - 1, end: ed.end})
			ed.start = ed.end
			ed.dot = ed.end
			ed.dirty = true
		}
	}
	return nil
}

func (ed *Editor) cmdDelete(undolines []string, action *[]undoAction) error {
	undolines = make([]string, ed.end-ed.start+1)
	copy(undolines, ed.Lines[ed.start-1:ed.end])
	*action = append(*action, undoAction{typ: undoAdd, start: ed.start, end: ed.start - 1, d: ed.end + len(undolines) - 1, lines: undolines})
	ed.Lines = append(ed.Lines[:ed.start-1], ed.Lines[ed.end:]...)

	if ed.start > len(ed.Lines) {
		ed.start = len(ed.Lines)
	}
	ed.end = ed.start
	ed.dot = ed.start
	ed.dirty = true
	return nil
}

func (ed *Editor) cmdEdit(unconditionally bool) error {
	ed.tok = ed.s.Scan()
	ed.skipWhitespace()
	var fname = ed.scanString()
	if !unconditionally && ed.dirty {
		ed.dirty = false
		return ErrFileModified
	}
	lines, err := ed.readFile(fname, true, true)
	if err != nil {
		return err
	}
	ed.Lines = lines
	return nil
}

func (ed *Editor) cmdFilename() error {
	if ed.s.Pos().Offset != 1 {
		return ErrUnexpectedAddress
	}
	if ed.tok = ed.s.Scan(); ed.tok == scanner.EOF {
		if ed.path == "" {
			return ErrNoFileName
		}
		fmt.Fprintln(ed.err, ed.path)
		return nil
	}
	if ed.tok = ed.s.Scan(); ed.tok == '!' {
		return ErrInvalidRedirection
	}
	var fname = ed.scanString()
	if fname == "" {
		return ErrNoFileName
	}
	ed.path = fname
	fmt.Fprintln(ed.err, ed.path)
	return nil
}

func (ed *Editor) cmdGlobal(interactive, inverted bool) error {
	if ed.g {
		return ErrCannotNestGlobal
	}
	if ed.s.Pos().Offset == 1 {
		ed.start = 1
		ed.end = len(ed.Lines)
	}
	ed.tok = ed.s.Scan()
	var delim = ed.tok
	ed.tok = ed.s.Scan()
	if delim == ' ' || delim == scanner.EOF {
		return ErrInvalidPatternDelim
	}
	var s, e, search = ed.start, ed.end, ed.scanStringUntil(delim)
	if ed.tok == delim {
		ed.tok = ed.s.Scan()
	}
	var cmd = ed.scanString()
	if cmd != "" {
		if cmd[:len(cmd)-1] == string(delim) {
			cmd = cmd[:len(cmd)-1]
		}
	}
	if cmd == "" && !interactive {
		cmd = "p"
	}
	ed.globalUndo = []undoAction{}
	ed.g = true
	defer func() {
		ed.g = false
		ed.undohist = append(ed.undohist, ed.globalUndo)
	}()
	for idx := s - 1; idx < e; idx++ {
		match, err := regexp.MatchString(search, ed.Lines[idx])
		if err != nil {
			return err
		}
		if (!match && !inverted) || (inverted && match) {
			continue
		}
		ed.start = idx + 1
		ed.end = ed.start
		ed.dot = ed.end
		if interactive {
			fmt.Fprintln(ed.out, ed.Lines[idx])
			r := bufio.NewReader(ed.in)
		loop:
			for {
				select {
				case <-ed.sigintch:
					fmt.Fprintln(ed.err, ErrDefault)
					break loop
				default:
					line, err := r.ReadString('\n')
					line = line[:len(line)-1]
					if err != nil {
						return err
					}
					cmd = line
					switch cmd {
					case "":
						continue
					case "&":
						cmd = ed.globalCmd
					}
				}
			}
		}
		ed.readInput(strings.NewReader(cmd))
		if err := ed.doCommand(); err != nil {
			return err
		}
		if e > len(ed.Lines) {
			e = len(ed.Lines)
		}
		ed.globalCmd = cmd
	}
	return nil
}

func (ed *Editor) cmdToggleError() error {
	ed.printErrors = !ed.printErrors
	return nil
}

func (ed *Editor) cmdExplainError() error {
	if ed.error != nil {
		fmt.Fprintln(ed.err, ed.error)
	}
	return ed.error
}

func (ed *Editor) cmdInsert(action *[]undoAction) error {
	d := ed.end
	r := bufio.NewReader(ed.in)
loop:
	for {
		select {
		case <-ed.sigintch:
			fmt.Fprintln(ed.err, ErrDefault)
			break loop
		default:
			line, err := r.ReadString('\n')
			if err != nil {
				break loop
			}
			line = line[:len(line)-1]
			if line == "." {
				break loop
			}
			if ed.end-1 < 0 {
				ed.end++
			}
			if ed.end > len(ed.Lines) {
				ed.Lines = append(ed.Lines, line)
			} else {
				ed.Lines = append(ed.Lines[:ed.end], ed.Lines[ed.end-1:]...)
				ed.Lines[ed.end-1] = line
			}
			ed.dirty = true
			*action = append(*action, undoAction{typ: undoDelete, start: ed.end - 1, end: ed.end, d: d})
			ed.end++
		}
	}
	ed.end--
	ed.start = ed.end
	ed.dot = ed.end
	return nil
}

func (ed *Editor) cmdJoin(undolines []string, action *[]undoAction) error {
	if ed.end == ed.start {
		ed.end++
	}
	if ed.end > len(ed.Lines) {
		return ErrInvalidAddress
	}
	undolines = make([]string, ed.end-ed.start+1)
	copy(undolines, ed.Lines[ed.start-1:ed.end])
	var joined = strings.Join(ed.Lines[ed.start-1:ed.end], "")
	*action = append(*action, undoAction{typ: undoAdd, start: ed.end - len(undolines) + 1, end: ed.end - len(undolines), d: ed.end + len(undolines), lines: undolines})
	var result []string = append(append([]string{}, ed.Lines[:ed.start-1]...), joined)
	ed.Lines = append(result, ed.Lines[ed.end:]...)
	*action = append(*action, undoAction{typ: undoDelete, start: ed.start - 1, end: ed.start})
	ed.end = ed.start
	ed.dot = ed.start
	ed.dirty = true
	return nil
}

func (ed *Editor) cmdMark() error {
	ed.tok = ed.s.Scan()
	var r rune = ed.tok
	ed.tok = ed.s.Scan()
	if r == scanner.EOF || !unicode.IsLower(r) {
		return ErrInvalidMark
	}
	var mark int = int(r) - 'a'
	if mark < 0 || mark > len(ed.mark) {
		return ErrInvalidMark
	}
	ed.mark[mark] = ed.end
	return nil
}

func (ed *Editor) cmdMove(undolines []string, action *[]undoAction) error {
	ed.tok = ed.s.Scan()
	dst, err := ed.scanNumber()
	if err != nil {
		ed.start = ed.dot
		ed.end = ed.dot
		return ErrInvalidCmdSuffix
	}
	if dst < 0 || dst > len(ed.Lines) {
		return ErrDestinationExpected
	}
	if ed.start <= dst && dst < ed.end {
		return ErrInvalidAddress
	}
	lines := make([]string, ed.end-ed.start+1)
	copy(lines, ed.Lines[ed.start-1:ed.end])
	*action = append(*action, undoAction{typ: undoAdd, start: ed.start, end: ed.start - 1, d: ed.end + 1, lines: lines})
	ed.Lines = append(ed.Lines[:ed.start-1], ed.Lines[ed.end:]...)
	if dst-len(lines) < 0 {
		dst = len(lines) + dst
	}
	ed.Lines = append(ed.Lines[:dst-len(lines)], append(lines, ed.Lines[dst-len(lines):]...)...)
	undolines = make([]string, len(lines))
	copy(undolines, ed.Lines[dst-len(lines):dst])
	*action = append(*action, undoAction{typ: undoDelete, start: dst - len(lines), end: dst})
	ed.end = dst
	ed.start = dst
	ed.dot = dst
	ed.dirty = true
	return nil
}

func (ed *Editor) cmdPrint() error {
	var numbers, unambiguous bool
check:
	for {
		switch ed.tok {
		case 'p':
			ed.tok = ed.s.Scan()
		case 'n':
			numbers = true
			ed.tok = ed.s.Scan()
		case 'l':
			unambiguous = true
			ed.tok = ed.s.Scan()
		default:
			break check
		}
	}
	for i := ed.start - 1; i < ed.end; i++ {
		if i < 0 {
			continue
		}
		if numbers {
			fmt.Fprintf(ed.out, "%d\t", i+1)
		}
		if unambiguous {
			var q = strconv.QuoteToASCII(ed.Lines[i])
			fmt.Fprintf(ed.out, "%s$\n", q[1:len(q)-1])
		} else {
			fmt.Fprintln(ed.out, ed.Lines[i])
		}
	}
	return nil
}

func (ed *Editor) cmdTogglePrompt() error {
	ed.showPrompt = !ed.showPrompt
	if ed.prompt == "" {
		ed.prompt = DefaultPrompt
	}
	return nil
}

func (ed *Editor) cmdQuit(unconditionally bool) error {
	if !unconditionally && ed.dirty {
		ed.dirty = false
		return ErrFileModified
	}
	os.Exit(0)
	return nil
}

func (ed *Editor) cmdRead(action *[]undoAction) error {
	ed.tok = ed.s.Scan()
	ed.skipWhitespace()
	var fname = ed.scanString()
	lines, err := ed.readFile(fname, false, true)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if len(ed.Lines) < len(lines) {
			ed.Lines = append(ed.Lines, line)
		} else {
			ed.Lines = append(ed.Lines[:ed.end], append([]string{line}, ed.Lines[ed.end:]...)...)
		}
		ed.dirty = true
		ed.end++
		*action = append(*action, undoAction{typ: undoDelete, start: ed.end - 1, end: ed.end})
	}
	ed.start = ed.end
	ed.dot = ed.end
	return nil
}

func (ed *Editor) cmdSubstitute(undolines []string, action *[]undoAction) error {
	ed.tok = ed.s.Scan()
	var (
		search, repl string
		mod          rune
		re           *regexp.Regexp
	)
	ed.tok = ed.s.Scan()
	if ed.tok == scanner.EOF {
		if ed.search == "" && ed.replacestr == "" {
			return ErrNoPreviousSub
		}
		search = ed.search
		repl = ed.replacestr
	} else {
		search = ed.scanStringUntil('/')
	}
	ed.tok = ed.s.Scan()
	if ed.tok != scanner.EOF {
		repl = ed.scanStringUntil('/')
		ed.tok = ed.s.Scan()
	}
	if repl == "%" && ed.replacestr != "" {
		repl = ed.replacestr
	}
	if ed.tok != scanner.EOF {
		mod = ed.tok
	}
	re, err := regexp.Compile(search)
	if err != nil {
		return ErrNoMatch
	}
	var (
		all bool = (mod == 'g')
		n   int  = 1
		N   int  = 1
	)
	if unicode.IsDigit(mod) {
		num, err := strconv.Atoi(string(mod))
		if err != nil {
			return ErrInvalidCmdSuffix
		}
		n = num
		N = num
	}
	var (
		match bool
		s     int = ed.start - 1
		e     int = ed.end
		d     int
		first bool = true
	)
	for i := e - 1; i >= s; i-- {
		n = N
		if re.MatchString(ed.Lines[i]) {
			if first {
				d = i
				first = false
			}
			match = true
			undolines = make([]string, 1)
			copy(undolines, []string{ed.Lines[i]})
			submatch := re.FindAllStringSubmatch(ed.Lines[i], -1)
			// TODO: Fix submatches
			ed.Lines[i] = re.ReplaceAllStringFunc(ed.Lines[i], func(s string) string {
				var cs scanner.Scanner
				cs.Init(strings.NewReader(repl))
				cs.Mode = scanner.ScanChars
				cs.Whitespace ^= scanner.GoWhitespace
				var (
					prepl string
					ctok  rune = cs.Scan()
				)
				for ctok != scanner.EOF {
					if ctok != '\\' && cs.Peek() == '&' {
						prepl += string(ctok)
						ctok = cs.Scan()
						_ = ctok
						ctok = cs.Scan()
						prepl += s
						continue
					} else if ctok == '\\' && unicode.IsDigit(cs.Peek()) {
						ctok = cs.Scan()
						n, err := strconv.Atoi(string(ctok))
						if err == nil && n-1 >= 0 && n-1 < len(submatch[0]) {
							ctok = cs.Scan()
							prepl += submatch[0][n-1]
						}
						continue
					}
					prepl += string(ctok)
					ctok = cs.Scan()
				}
				n--
				if all || n == 0 {
					return prepl
				}
				return s
			})
			ed.start = d + 1
			ed.end = ed.start
			ed.dot = ed.start
			*action = append(*action, undoAction{typ: undoAdd, start: i + 1, end: i, d: e, lines: undolines})
			*action = append(*action, undoAction{typ: undoDelete, start: i, end: i + 1, lines: []string{ed.Lines[i]}})
			ed.dirty = true
		}
	}
	if !match {
		return ErrNoMatch
	}
	return nil
}

func (ed *Editor) cmdTransfer(action *[]undoAction) error {
	ed.tok = ed.s.Scan()
	if ed.tok == scanner.EOF {
		return ErrDestinationExpected
	}
	dst, err := ed.scanNumber()
	if err != nil {
		return ErrDestinationExpected
	}
	if ed.start-1 < 0 || ed.end > len(ed.Lines) || dst > len(ed.Lines) || dst < 0 {
		return ErrInvalidAddress
	}
	lines := make([]string, ed.end-ed.start+1)
	copy(lines, ed.Lines[ed.start-1:ed.end])
	ed.Lines = append(ed.Lines[:dst], append(lines, ed.Lines[dst:]...)...)
	ed.end = dst + len(lines)
	*action = append(*action, undoAction{typ: undoDelete, start: dst, end: dst + len(lines), d: dst + 1})
	ed.start = ed.end
	ed.dot = ed.end
	ed.dirty = true
	return nil
}

func (ed *Editor) cmdUndo() error {
	if ed.s.Pos().Offset != 1 {
		return ErrUnexpectedAddress
	}
	if ed.s.Peek() != scanner.EOF {
		return ErrInvalidCmdSuffix
	}
	return ed.undo()
}

func (ed *Editor) cmdWrite(append bool) error {
	var (
		quit bool
		full = (ed.s.Pos().Offset == 1)
	)
	ed.tok = ed.s.Scan()
	if !append {
		if ed.tok == 'q' {
			ed.tok = ed.s.Scan()
			quit = true
		}
	}
	if ed.tok == ' ' {
		ed.tok = ed.s.Scan()
	}
	var fname = ed.scanString()
	if fname == "" && ed.path == "" {
		return ErrNoFileName
	}
	if fname == "" {
		fname = ed.path
	}
	var s, e int = ed.start, ed.end
	if full {
		s = 1
		e = len(ed.Lines)
	}
	var err error
	if append {
		err = ed.appendFile(s, e, fname)
	} else {
		err = ed.writeFile(s, e, fname)
	}
	if quit {
		os.Exit(0)
	}
	return err
}

func (ed *Editor) cmdScroll() error {
	ed.tok = ed.s.Scan()
	scroll, err := ed.scanNumber()
	if err != nil || scroll == 0 {
		scroll = ed.scroll
	}
	ed.start = ed.end - 1
	ed.end += scroll
	if ed.end > len(ed.Lines) {
		ed.end = len(ed.Lines)
	}
	for ; ed.start < ed.end; ed.start++ {
		fmt.Fprintln(ed.out, ed.Lines[ed.start])
	}
	ed.dot = ed.start + 1
	ed.scroll = scroll
	return nil
}

func (ed *Editor) cmdPrintLineNumber() error {
	fmt.Fprintln(ed.out, len(ed.Lines))
	return nil
}

func (ed *Editor) cmdExecute() error {
	ed.tok = ed.s.Scan()
	ed.skipWhitespace()
	var buf = ed.scanString()
	output, err := ed.readFile("!"+buf, false, false)
	if err != nil {
		return err
	}
	for i := range output {
		fmt.Fprintln(ed.err, output[i])
	}
	fmt.Fprintln(ed.err, "!")
	return nil
}
