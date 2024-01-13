package main

import (
	"regexp"
	"text/scanner"
	"unicode"
)

// nextAddress attempts to extract the next valid address. It navigates
// through the user input, identifying and returning the next address
// based on predefined criteria, for more info see the ed(1) man page.
func (ed *Editor) nextAddress() (int, error) {
	ed.addr = ed.Dot
	var mod rune
	var first bool
	for first = true; ; first = false {
		for {
			switch ed.tok {
			case ' ':
				fallthrough
			case '\t':
				fallthrough
			case '\n':
				fallthrough
			case '\r':
				ed.tok = ed.s.Scan()
				continue
			}
			break
		}
		switch {
		case ed.tok == '.':
			fallthrough
		case ed.tok == '$':
			if !first {
				return 0, ErrInvalidAddress
			}
			if ed.tok == '.' {
				ed.addr = ed.Dot
			} else {
				ed.addr = len(ed.Lines)
			}
			ed.tok = ed.s.Scan()
		case ed.tok == '?':
			fallthrough
		case ed.tok == '/':
			var mod rune = ed.tok
			ed.tok = ed.s.Scan()
			var search string = ed.scanStringUntil(mod)
			if ed.tok == mod {
				ed.tok = ed.s.Scan()
			}
			if search == "" {
				if ed.search == "" {
					return 0, ErrNoPrevPattern
				}
				search = ed.search
			}
			ed.search = search
			var s int = 0 // ed.End - 1
			var e = len(ed.Lines)
			if mod == '?' {
				s = ed.End - 1
				e = 0
			}
			for i := s; i != e; {
				if i < 0 || i > len(ed.Lines) {
					return 0, ErrNoMatch
				}
				match, err := regexp.MatchString(search, ed.Lines[i])
				if err != nil {
					return 0, ErrNoMatch
				}
				if match {
					ed.addr = i + 1
					return ed.addr, nil
				}
				if mod == '/' {
					i++
				} else {
					i--
				}
			}
			return 0, ErrNoMatch
		case ed.tok == '\'':
			ed.tok = ed.s.Scan()
			var r rune = ed.tok
			ed.tok = ed.s.Scan()
			if r == scanner.EOF || !unicode.IsLower(r) {
				return 0, ErrInvalidMark
			}
			var mark int = int(r) - 'a'
			if mark < 0 || mark > len(ed.mark) {
				return 0, ErrInvalidMark
			}
			var maddr int = ed.mark[mark]
			if maddr <= 0 || maddr > len(ed.Lines) {
				return 0, ErrInvalidAddress
			}
			ed.addr = maddr
		case ed.tok == '+':
			fallthrough
		case ed.tok == '-':
			fallthrough
		case ed.tok == '^':
			mod = ed.tok
			ed.tok = ed.s.Scan()
			if !unicode.IsDigit(ed.tok) {
				switch mod {
				case '^', '-':
					ed.addr--
				case '+':
					ed.addr++
				}
			}
			fallthrough
		case unicode.IsDigit(ed.tok):
			if !first {
				return 0, ErrInvalidAddress
			}
			if unicode.IsDigit(ed.tok) {
				n, err := ed.scanNumber()
				if err != nil {
					return 0, ErrInvalidNumber
				}
				switch mod {
				case '^', '-':
					ed.addr -= n
				case '+':
					ed.addr += n
				default:
					ed.addr = n
				}
			}
		case ed.tok == ';':
			fallthrough
		case ed.tok == '%':
			fallthrough
		case ed.tok == ',':
			var r rune = ed.tok
			if first {
				ed.tok = ed.s.Scan()
				var err error
				n, err := ed.nextAddress()
				if err != nil {
					return 0, err
				}
				ed.addr = n
				if n == ed.Dot && ed.Start == ed.End {
					ed.Start = 1
					if r == ';' {
						ed.Start = ed.End
					}
					ed.Dot = len(ed.Lines)
					ed.addr = ed.Dot
					ed.End = ed.Dot
					ed.addrCount = 2
					return -1, nil
				}
				continue
			}
			fallthrough
		default:
			if ed.addr < 0 || ed.addr > len(ed.Lines) {
				return -1, ErrInvalidAddress
			}
			return ed.addr, nil
		}
	}
}

// DoRange parses user input to extract the specified line or range
// on which the user intends to execute commands.
func (ed *Editor) DoRange() error {
	var n int
	var err error
	ed.addrCount = 0
	ed.Start = ed.Dot
	ed.End = ed.Dot
	if ed.tok == scanner.EOF {
		goto end
	}
	for {
		n, err = ed.nextAddress()
		if n < 0 {
			break
		}
		if err != nil {
			return err
		}
		ed.addr = n
		ed.addrCount++
		ed.Start = ed.End
		ed.End = ed.addr
		if ed.tok != ',' && ed.tok != ';' {
			break
		} else if ed.s.Peek() == ';' {
			ed.Dot = ed.addr
		}
		if ed.tok == scanner.EOF {
			break
		}
	}
end:
	if ed.addrCount == 1 || ed.End != ed.addr {
		ed.Start = ed.End
	}
	ed.Dot = ed.End
	if err := ed.checkRange(); err != nil {
		return err
	}
	return nil
}
