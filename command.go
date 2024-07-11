package main

import (
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
	var ul []string
	var uo []undoOperation
	defer func() {
		if len(uo) > 0 {
			if !ed.g {
				ed.undohist = append(ed.undohist, uo)
			} else {
				ed.globalUndo = append(ed.globalUndo, uo...)
			}
		}
		if err != nil {
			ed.error = err
		}
	}()
	switch ed.tok {
	case 'a':
		for {
			line, err := ed.readInsert()
			if err != nil {
				ed.setupSignals()
				break
			}
			if line == "." {
				break
			}
			if len(ed.Lines) == ed.end {
				ed.Lines = append(ed.Lines, line)
			} else {
				ed.Lines = append(ed.Lines[:ed.end], append([]string{line}, ed.Lines[ed.end:]...)...)
			}
			ed.end++
			uo = append(uo, undoOperation{typ: undoDelete, start: ed.end - 1, end: ed.end})
			ed.start = ed.end
			ed.dot = ed.end
			ed.dirty = true
		}
		return nil
	case 'c':
		ul = make([]string, ed.end-ed.start+1)
		copy(ul, ed.Lines[ed.start-1:ed.end])
		uo = append(uo, undoOperation{typ: undoAdd, start: ed.start, end: ed.start - 1, d: ed.start + len(ul) - 1, lines: ul})
		ed.Lines = append(ed.Lines[:ed.start-1], ed.Lines[ed.end:]...)
		ed.end = ed.start - 1
		for {
			line, err := ed.readInsert()
			if err != nil {
				ed.setupSignals()
				break
			}
			if line == "." {
				break
			}
			if ed.end > len(ed.Lines) {
				ed.Lines = append(ed.Lines[:ed.end], line)
			} else {
				ed.Lines = append(ed.Lines[:ed.end], append([]string{line}, ed.Lines[ed.end:]...)...)
			}
			ed.end++
			uo = append(uo, undoOperation{typ: undoDelete, start: ed.end - 1, end: ed.end})
			ed.start = ed.end
			ed.dot = ed.end
			ed.dirty = true
		}
		return nil
	case 'd':
		ul = make([]string, ed.end-ed.start+1)
		copy(ul, ed.Lines[ed.start-1:ed.end])
		uo = append(uo, undoOperation{typ: undoAdd, start: ed.start, end: ed.start - 1, d: ed.end + len(ul) - 1, lines: ul})
		ed.Lines = append(ed.Lines[:ed.start-1], ed.Lines[ed.end:]...)
		if ed.start > len(ed.Lines) {
			ed.start = len(ed.Lines)
		}
		ed.end = ed.start
		ed.dot = ed.start
		ed.dirty = true
		return nil
	case 'E':
		fallthrough
	case 'e':
		var uc = (ed.tok == 'E')
		ed.tok = ed.s.Scan()
		ed.skipWhitespace()
		var fname = ed.scanString()
		if !uc && ed.dirty {
			ed.dirty = false
			return ErrFileModified
		}
		var lines []string
		lines, err = ed.readFile(fname, true, true)
		if err != nil {
			return err
		}
		ed.Lines = lines
		return nil
	case 'f':
		ed.tok = ed.s.Scan()
		if ed.tok == scanner.EOF {
			if ed.path == "" {
				return ErrNoFileName
			}
			fmt.Fprintf(ed.err, "%s\n", ed.path)
			return nil
		}
		ed.tok = ed.s.Scan()
		var fname = ed.scanString()
		if fname == "" {
			return ErrNoFileName
		}
		ed.path = fname
		fmt.Fprintf(ed.err, "%s\n", ed.path)
		return nil
	case 'V':
		fallthrough
	case 'G':
		fallthrough
	case 'v':
		fallthrough
	case 'g':
		if ed.g {
			return ErrCannotNestGlobal
		}
		var i = (ed.tok == 'G' || ed.tok == 'V')
		var v = (ed.tok == 'v' || ed.tok == 'V')
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
		var s, e = ed.start, ed.end
		var search = ed.scanStringUntil(delim)
		if ed.tok == delim {
			ed.tok = ed.s.Scan()
		}
		var cmd = ed.scanString()
		if cmd != "" {
			if cmd[:len(cmd)-1] == string(delim) {
				cmd = cmd[:len(cmd)-1]
			}
		}
		if cmd == "" && !i {
			cmd = "p"
		}
		ed.globalUndo = []undoOperation{}
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
			if (!match && !v) || (v && match) {
				continue
			}
			ed.start = idx + 1
			ed.end = ed.start
			ed.dot = ed.end
			if i {
				fmt.Fprintf(ed.out, "%s\n", ed.Lines[idx])
				line, err := ed.readInsert()
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
	case 'H':
		ed.printErrors = !ed.printErrors
		return nil
	case 'h':
		if ed.error != nil {
			fmt.Fprintf(ed.err, "%s\n", ed.error)
		}
		return nil
	case 'i':
		d := ed.end
		for {
			line, err := ed.readInsert()
			if err != nil {
				ed.setupSignals()
				break
			}
			if line == "." {
				break
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
			uo = append(uo, undoOperation{typ: undoDelete, start: ed.end - 1, end: ed.end, d: d})
			ed.end++
		}
		ed.end--
		ed.start = ed.end
		ed.dot = ed.end
		return nil
	case 'j':
		if ed.end == ed.start {
			ed.end++
		}
		if ed.end > len(ed.Lines) {
			return ErrInvalidAddress
		}
		ul = make([]string, ed.end-ed.start+1)
		copy(ul, ed.Lines[ed.start-1:ed.end])
		var joined = strings.Join(ed.Lines[ed.start-1:ed.end], "")
		uo = append(uo, undoOperation{typ: undoAdd, start: ed.end - len(ul) + 1, end: ed.end - len(ul), d: ed.end + len(ul), lines: ul})
		var result []string = append(append([]string{}, ed.Lines[:ed.start-1]...), joined)
		ed.Lines = append(result, ed.Lines[ed.end:]...)
		uo = append(uo, undoOperation{typ: undoDelete, start: ed.start - 1, end: ed.start})
		ed.end = ed.start
		ed.dot = ed.start
		ed.dirty = true
		return nil
	case 'k':
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
	case 'm':
		var dst int
		ed.tok = ed.s.Scan()
		dst, err = ed.scanNumber()
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
		uo = append(uo, undoOperation{typ: undoAdd, start: ed.start, end: ed.start - 1, d: ed.end + 1, lines: lines})
		ed.Lines = append(ed.Lines[:ed.start-1], ed.Lines[ed.end:]...)
		if dst-len(lines) < 0 {
			dst = len(lines) + dst
		}
		ed.Lines = append(ed.Lines[:dst-len(lines)], append(lines, ed.Lines[dst-len(lines):]...)...)
		ul = make([]string, len(lines))
		copy(ul, ed.Lines[dst-len(lines):dst])
		uo = append(uo, undoOperation{typ: undoDelete, start: dst - len(lines), end: dst})
		ed.end = dst
		ed.start = dst
		ed.dot = dst
		ed.dirty = true
		return nil
	case 'l':
		fallthrough
	case 'n':
		fallthrough
	case 'p':
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
				fmt.Fprintf(ed.out, "%s\n", ed.Lines[i])
			}
		}
		return nil
	case 'P':
		ed.showPrompt = !ed.showPrompt
		if ed.prompt == "" {
			ed.prompt = defaultPrompt
		}
		return nil
	case 'q':
		fallthrough
	case 'Q':
		if ed.tok == 'q' && ed.dirty {
			ed.dirty = false
			return ErrFileModified
		}
		os.Exit(0)
		return nil
	case 'r':
		ed.tok = ed.s.Scan()
		ed.skipWhitespace()
		var fname = ed.scanString()
		var lines []string
		lines, err = ed.readFile(fname, false, true)
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
			uo = append(uo, undoOperation{typ: undoDelete, start: ed.end - 1, end: ed.end})
		}
		ed.start = ed.end
		ed.dot = ed.end
		return nil
	case 's':
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
		re, err = regexp.Compile(search)
		if err != nil {
			return ErrNoMatch
		}
		var all bool = (mod == 'g')
		var n int = 1
		var N int = 1
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
				ul = make([]string, 1)
				copy(ul, []string{ed.Lines[i]})
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
				uo = append(uo, undoOperation{typ: undoAdd, start: i + 1, end: i, d: e, lines: ul})
				uo = append(uo, undoOperation{typ: undoDelete, start: i, end: i + 1, lines: []string{ed.Lines[i]}})
				ed.dirty = true
			}
		}
		if !match {
			return ErrNoMatch
		}
		return nil
	case 't':
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
		var lines = make([]string, ed.end-ed.start+1)
		copy(lines, ed.Lines[ed.start-1:ed.end])
		ed.Lines = append(ed.Lines[:dst], append(lines, ed.Lines[dst:]...)...)
		ed.end = dst + len(lines)
		uo = append(uo, undoOperation{typ: undoDelete, start: dst, end: dst + len(lines), d: dst + 1})
		ed.start = ed.end
		ed.dot = ed.end
		ed.dirty = true
		return nil
	case 'u':
		if ed.s.Pos().Offset != 1 {
			return ErrUnexpectedAddress
		}
		if ed.s.Peek() != scanner.EOF {
			return ErrInvalidCmdSuffix
		}
		return ed.undo()
	case 'W':
		fallthrough
	case 'w':
		var (
			quit bool
			r    = ed.tok
			full = (ed.s.Pos().Offset == 1)
		)
		ed.tok = ed.s.Scan()
		if r == 'w' {
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
		if r == 'w' {
			err = ed.writeFile(s, e, fname)
		} else {
			err = ed.appendFile(s, e, fname)
		}
		if quit {
			os.Exit(0)
		}
		return err
	case 'z':
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
			fmt.Fprintf(ed.out, "%s\n", ed.Lines[ed.start])
		}
		ed.dot = ed.start + 1
		ed.scroll = scroll
		return nil
	case '=':
		fmt.Fprintf(ed.out, "%d\n", len(ed.Lines))
		return nil
	case '!':
		ed.tok = ed.s.Scan()
		ed.skipWhitespace()
		var buf = ed.scanString()
		output, err := ed.readFile("!"+buf, false, false)
		if err != nil {
			return err
		}
		for i := range output {
			fmt.Fprintf(ed.err, "%s\n", output[i])
		}
		fmt.Fprintln(ed.err, "!")
		return nil
	case 0:
		fallthrough
	case scanner.EOF:
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
		fmt.Fprintf(ed.out, "%s\n", ed.Lines[ed.end-1])
		return err
	default:
		return ErrUnknownCmd
	}
}
