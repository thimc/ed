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

// DoCommand executes a command on a range defined by start and end.
func (ed *Editor) DoCommand() (err error) {
	var ul []string
	var uo []undoOp
	defer func() {
		if len(uo) > 0 {
			if !ed.g {
				ed.undo = append(ed.undo, uo)
			} else {
				ed.globalUndo = append(ed.globalUndo, uo...)
			}
		}
		if err != nil {
			ed.Error = err
		}
	}()
	switch ed.tok {
	case 'a':
		for {
			line, err := ed.ReadInsert()
			if err != nil {
				ed.setupSignals()
				break
			}
			if line == "." {
				break
			}
			if len(ed.Lines) == ed.End {
				ed.Lines = append(ed.Lines, line)
			} else {
				ed.Lines = append(ed.Lines[:ed.End], append([]string{line}, ed.Lines[ed.End:]...)...)
			}
			ed.End++
			uo = append(uo, undoOp{action: undoDelete, start: ed.End - 1, end: ed.End})
			ed.Start = ed.End
			ed.Dot = ed.End
			ed.Dirty = true
		}
		return nil
	case 'c':
		ul = make([]string, ed.End-ed.Start+1)
		copy(ul, ed.Lines[ed.Start-1:ed.End])
		uo = append(uo, undoOp{action: undoAdd, start: ed.Start, end: ed.Start - 1, d: ed.Start + len(ul) - 1, lines: ul})
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		ed.End = ed.Start - 1
		for {
			line, err := ed.ReadInsert()
			if err != nil {
				ed.setupSignals()
				break
			}
			if line == "." {
				break
			}
			if ed.End > len(ed.Lines) {
				ed.Lines = append(ed.Lines[:ed.End], line)
			} else {
				ed.Lines = append(ed.Lines[:ed.End], append([]string{line}, ed.Lines[ed.End:]...)...)
			}
			ed.End++
			uo = append(uo, undoOp{action: undoDelete, start: ed.End - 1, end: ed.End})
			ed.Start = ed.End
			ed.Dot = ed.End
			ed.Dirty = true
		}
		return nil
	case 'd':
		ul = make([]string, ed.End-ed.Start+1)
		copy(ul, ed.Lines[ed.Start-1:ed.End])
		uo = append(uo, undoOp{action: undoAdd, start: ed.Start, end: ed.Start - 1, d: ed.End + len(ul) - 1, lines: ul})
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		if ed.Start > len(ed.Lines) {
			ed.Start = len(ed.Lines)
		}
		ed.End = ed.Start
		ed.Dot = ed.Start
		ed.Dirty = true
		return nil
	case 'E':
		fallthrough
	case 'e':
		var uc bool = (ed.tok == 'E')
		ed.tok = ed.s.Scan()
		ed.skipWhitespace()
		var fname string = ed.scanString()
		if !uc && ed.Dirty {
			ed.Dirty = false
			return ErrFileModified
		}
		var lines []string
		lines, err = ed.ReadFile(fname, true, true)
		if err != nil {
			return err
		}
		ed.Lines = lines
		return nil
	case 'f':
		ed.tok = ed.s.Scan()
		if ed.tok == scanner.EOF {
			if ed.Path == "" {
				return ErrNoFileName
			}
			fmt.Fprintf(ed.err, "%s\n", ed.Path)
			return nil
		}
		ed.tok = ed.s.Scan()
		var fname string = ed.scanString()
		if fname == "" {
			return ErrNoFileName
		}
		ed.Path = fname
		fmt.Fprintf(ed.err, "%s\n", ed.Path)
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
		var i bool = (ed.tok == 'G' || ed.tok == 'V')
		var v bool = (ed.tok == 'v' || ed.tok == 'V')
		if ed.s.Pos().Offset == 1 {
			ed.Start = 1
			ed.End = len(ed.Lines)
		}
		ed.tok = ed.s.Scan()
		var delim rune = ed.tok
		ed.tok = ed.s.Scan()
		if delim == ' ' || delim == scanner.EOF {
			return ErrInvalidPatternDelim
		}
		var s int = ed.Start
		var e int = ed.End
		var search string = ed.scanStringUntil(delim)
		if ed.tok == delim {
			ed.tok = ed.s.Scan()
		}
		var cmd string = ed.scanString()
		if cmd != "" {
			if cmd[:len(cmd)-1] == string(delim) {
				cmd = cmd[:len(cmd)-1]
			}
		}
		if cmd == "" && !i {
			cmd = "p"
		}
		ed.globalUndo = []undoOp{}
		ed.g = true
		defer func() {
			ed.g = false
			ed.undo = append(ed.undo, ed.globalUndo)
		}()
		for idx := s - 1; idx < e; idx++ {
			match, err := regexp.MatchString(search, ed.Lines[idx])
			if err != nil {
				return err
			}
			if (!match && !v) || (v && match) {
				continue
			}
			ed.Start = idx + 1
			ed.End = ed.Start
			ed.Dot = ed.End
			if i {
				fmt.Fprintf(ed.out, "%s\n", ed.Lines[idx])
				line, err := ed.ReadInsert()
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
			ed.ReadInput(strings.NewReader(cmd))
			if err := ed.DoCommand(); err != nil {
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
		if ed.Error != nil {
			fmt.Fprintf(ed.err, "%s\n", ed.Error)
		}
		return nil
	case 'i':
		d := ed.End
		for {
			line, err := ed.ReadInsert()
			if err != nil {
				ed.setupSignals()
				break
			}
			if line == "." {
				break
			}
			if ed.End-1 < 0 {
				ed.End++
			}
			if ed.End > len(ed.Lines) {
				ed.Lines = append(ed.Lines, line)
			} else {
				ed.Lines = append(ed.Lines[:ed.End], ed.Lines[ed.End-1:]...)
				ed.Lines[ed.End-1] = line
			}
			ed.Dirty = true
			uo = append(uo, undoOp{action: undoDelete, start: ed.End - 1, end: ed.End, d: d})
			ed.End++
		}
		ed.End--
		ed.Start = ed.End
		ed.Dot = ed.End
		return nil
	case 'j':
		if ed.End == ed.Start {
			ed.End++
		}
		if ed.End > len(ed.Lines) {
			return ErrInvalidAddress
		}
		ul = make([]string, ed.End-ed.Start+1)
		copy(ul, ed.Lines[ed.Start-1:ed.End])
		var joined string = strings.Join(ed.Lines[ed.Start-1:ed.End], "")
		uo = append(uo, undoOp{action: undoAdd, start: ed.End - len(ul) + 1, end: ed.End - len(ul), d: ed.End + len(ul), lines: ul})
		var result []string = append(append([]string{}, ed.Lines[:ed.Start-1]...), joined)
		ed.Lines = append(result, ed.Lines[ed.End:]...)
		uo = append(uo, undoOp{action: undoDelete, start: ed.Start - 1, end: ed.Start})
		ed.End = ed.Start
		ed.Dot = ed.Start
		ed.Dirty = true
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
		ed.mark[mark] = ed.End
		return nil
	case 'm':
		var err error
		var dst int
		ed.tok = ed.s.Scan()
		dst, err = ed.scanNumber()
		if err != nil {
			ed.Start = ed.Dot
			ed.End = ed.Dot
			return ErrInvalidCmdSuffix
		}
		if dst < 0 || dst > len(ed.Lines) {
			return ErrDestinationExpected
		}
		if ed.Start <= dst && dst < ed.End {
			return ErrInvalidAddress
		}
		lines := make([]string, ed.End-ed.Start+1)
		copy(lines, ed.Lines[ed.Start-1:ed.End])
		uo = append(uo, undoOp{action: undoAdd, start: ed.Start, end: ed.Start - 1, d: ed.End + 1, lines: lines})
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		if dst-len(lines) < 0 {
			dst = len(lines) + dst
		}
		ed.Lines = append(ed.Lines[:dst-len(lines)], append(lines, ed.Lines[dst-len(lines):]...)...)
		ul = make([]string, len(lines))
		copy(ul, ed.Lines[dst-len(lines):dst])
		uo = append(uo, undoOp{action: undoDelete, start: dst - len(lines), end: dst})
		ed.End = dst
		ed.Start = dst
		ed.Dot = dst
		ed.Dirty = true
		return nil
	case 'l':
		fallthrough
	case 'n':
		fallthrough
	case 'p':
		for i := ed.Start - 1; i < ed.End; i++ {
			if i < 0 {
				continue
			}
			switch ed.tok {
			case 'l':
				var q string = strconv.QuoteToASCII(ed.Lines[i])
				fmt.Fprintf(ed.out, "%s$\n", q[1:len(q)-1])
			case 'n':
				fmt.Fprintf(ed.out, "%d\t%s\n", i+1, ed.Lines[i])
			case 'p':
				fmt.Fprintf(ed.out, "%s\n", ed.Lines[i])
			}
		}
		return nil
	case 'P':
		ed.showPrompt = !ed.showPrompt
		if ed.Prompt == "" {
			ed.Prompt = defaultPrompt
		}
		return nil
	case 'q':
		fallthrough
	case 'Q':
		if ed.tok == 'q' && ed.Dirty {
			ed.Dirty = false
			return ErrFileModified
		}
		os.Exit(0)
		return nil
	case 'r':
		ed.tok = ed.s.Scan()
		ed.skipWhitespace()
		var fname string = ed.scanString()
		var lines []string
		lines, err = ed.ReadFile(fname, false, true)
		if err != nil {
			return err
		}
		for _, line := range lines {
			if len(ed.Lines) < len(lines) {
				ed.Lines = append(ed.Lines, line)
			} else {
				ed.Lines = append(ed.Lines[:ed.End], append([]string{line}, ed.Lines[ed.End:]...)...)
			}
			ed.Dirty = true
			ed.End++
			uo = append(uo, undoOp{action: undoDelete, start: ed.End - 1, end: ed.End})
		}
		ed.Start = ed.End
		ed.Dot = ed.End
		return nil
	case 's':
		ed.tok = ed.s.Scan()
		var search, repl string
		var mod rune
		var re *regexp.Regexp
		var err error
		ed.tok = ed.s.Scan()
		if ed.tok == scanner.EOF {
			if ed.search == "" && ed.replacestr == "" {
				return ErrNoPreviousSub
			}
			search = ed.search
			repl = ed.replacestr
		}
		search = ed.scanStringUntil('/')
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
		var match bool
		var s int = ed.Start - 1
		var e int = ed.End
		var d int
		var first bool = true
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
					var prepl string
					var ctok rune = cs.Scan()
					for ctok != scanner.EOF {
						if ctok != '\\' && cs.Peek() == '&' {
							prepl += string(ctok)
							ctok = cs.Scan()
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
					if all {
						n = 0
					}
					if n == 0 {
						return prepl
					}
					return s
				})
				ed.Start = d + 1
				ed.End = ed.Start
				ed.Dot = ed.Start
				uo = append(uo, undoOp{action: undoAdd, start: i + 1, end: i, d: e, lines: ul})
				uo = append(uo, undoOp{action: undoDelete, start: i, end: i + 1, lines: []string{ed.Lines[i]}})
				ed.Dirty = true
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
		if ed.Start-1 < 0 || ed.End > len(ed.Lines) || dst > len(ed.Lines) || dst < 0 {
			return ErrInvalidAddress
		}
		var lines []string = make([]string, ed.End-ed.Start+1)
		copy(lines, ed.Lines[ed.Start-1:ed.End])
		ed.Lines = append(ed.Lines[:dst], append(lines, ed.Lines[dst:]...)...)
		ed.End = dst + len(lines)
		uo = append(uo, undoOp{action: undoDelete, start: dst, end: dst + len(lines), d: dst + 1})
		ed.Start = ed.End
		ed.Dot = ed.End
		ed.Dirty = true
		return nil
	case 'u':
		if ed.s.Pos().Offset != 1 {
			return ErrUnexpectedAddress
		}
		if ed.s.Peek() != scanner.EOF {
			return ErrInvalidCmdSuffix
		}
		return ed.Undo()
	case 'W':
		fallthrough
	case 'w':
		var quit bool
		var r rune = ed.tok
		var full bool = (ed.s.Pos().Offset == 1)
		ed.tok = ed.s.Scan()
		if r == 'w' {
			if ed.tok == 'q' {
				ed.tok = ed.s.Scan()
				quit = true
			}
		} else {
		}
		if ed.tok == ' ' {
			ed.tok = ed.s.Scan()
		}
		var fname string = ed.scanString()
		if fname == "" && ed.Path == "" {
			return ErrNoFileName
		}
		if fname == "" {
			fname = ed.Path
		}
		var s int = ed.Start
		var e int = ed.End
		if full {
			s = 1
			e = len(ed.Lines)
		}
		var err error
		if r == 'w' {
			err = ed.WriteFile(s, e, fname)
		} else {
			err = ed.AppendFile(s, e, fname)
		}
		if quit {
			os.Exit(0)
		}
		return err
	case 'z':
		ed.tok = ed.s.Scan()
		var err error
		var scroll int
		scroll, err = ed.scanNumber()
		if err != nil || scroll == 0 {
			scroll = ed.scroll
		}
		if ed.End-1 < 0 {
			return ErrInvalidAddress
		}
		var s int = ed.End - 1
		var e int = ed.End + scroll
		if e > len(ed.Lines) {
			e = len(ed.Lines)
		}
		for i := s; i < e; i++ {
			fmt.Fprintf(ed.out, "%s\n", ed.Lines[i])
			ed.End = i
			ed.Start = i
		}
		ed.Start++
		ed.End++
		ed.Dot = ed.Start + 1
		ed.scroll = scroll
		return nil
	case '=':
		fmt.Fprintf(ed.out, "%d\n", len(ed.Lines))
		return nil
	case '!':
		ed.tok = ed.s.Scan()
		ed.skipWhitespace()
		var buf string = ed.scanString()
		var output []string
		output, err = ed.ReadFile("!"+buf, false, false)
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
			ed.Dot++
			if ed.Dot >= len(ed.Lines) {
				ed.Dot = len(ed.Lines)
				err = ErrInvalidAddress
			}
			ed.Start = ed.Dot
			ed.End = ed.Dot
		}
		if err == nil {
			if ed.End-1 < 0 {
				return ErrInvalidAddress
			}
			fmt.Fprintf(ed.out, "%s\n", ed.Lines[ed.End-1])
		}
		return err
	default:
		return ErrUnknownCmd
	}
}
