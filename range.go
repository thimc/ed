package main

import (
	"regexp"
	"text/scanner"
	"unicode"
)

func (ed *Editor) Range() (int, error) {
	ed.addr = ed.Dot
	var mod rune
	var first bool
	for first = true; ; first = false {
		for {
			switch ed.token() {
			case ' ':
				fallthrough
			case '\t':
				fallthrough
			case '\n':
				fallthrough
			case '\r':
				ed.nextToken()
				continue
			}
			break
		}
		switch {
		case ed.token() == '.':
			fallthrough
		case ed.token() == '$':
			if !first {
				return 0, ErrInvalidAddress
			}
			if ed.token() == '.' {
				ed.addr = ed.Dot
			} else {
				ed.addr = len(ed.Lines)
			}
			ed.nextToken()
		case ed.token() == '?':
			fallthrough
		case ed.token() == '/':
			var mod rune = ed.token()
			ed.nextToken()
			var search string = ed.scanString()
			if search == string(mod) || search == "" {
				if ed.search == "" {
					return 0, ErrNoPrevPattern
				}
				search = ed.search
			} else if search[len(search)-1] == byte(mod) {
				search = search[:len(search)-1]
			}
			ed.search = search
			var s int = ed.End - 1
			var e = len(ed.Lines)
			if mod == '?' {
				e = 0
			}
			ed.dump()
			for i := s; i != e; {
				if i < 0 || i > len(ed.Lines) {
					return 0, ErrNoMatch
				}
				match, err := regexp.MatchString(search, ed.Lines[i])
				if err != nil {
					return 0, err
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
		case ed.token() == '\'':
			ed.nextToken()
			var r rune = ed.token()
			ed.nextToken()
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
			ed.dump()
		case ed.token() == '+':
			fallthrough
		case ed.token() == '-':
			fallthrough
		case ed.token() == '^':
			mod = ed.token()
			ed.nextToken()
			if !unicode.IsDigit(ed.token()) {
				switch mod {
				case '^', '-':
					ed.addr--
				case '+':
					ed.addr++
				}
			}
			fallthrough
		case unicode.IsDigit(ed.token()):
			if !first {
				return 0, ErrInvalidAddress
			}
			if unicode.IsDigit(ed.token()) {
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
		case ed.token() == ';':
			fallthrough
		case ed.token() == '%':
			fallthrough
		case ed.token() == ',':
			var r rune = ed.token()
			if first {
				ed.nextToken()
				var err error
				n, err := ed.Range()
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

func (ed *Editor) DoRange() error {
	var n int
	var err error
	ed.addrCount = 0
	ed.Start = ed.Dot
	ed.End = ed.Dot
	if ed.token() == scanner.EOF {
		goto end
	}
	for {
		n, err = ed.Range()
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
		if ed.token() != ',' && ed.token() != ';' {
			break
		} else if ed.s.Peek() == ';' {
			ed.Dot = ed.addr
		}
		if ed.token() == scanner.EOF {
			break
		}
	}
end:
	if ed.addrCount == 1 || ed.End != ed.addr {
		ed.Start = ed.End
	}
	ed.Dot = ed.End
	if ed.token() == scanner.EOF && ed.s.Pos().Offset == 0 {
		ed.Dot++
		ed.Start = ed.Dot
		ed.End = ed.Dot
	}
	if err := ed.checkRange(); err != nil {
		return err
	}
	return nil
}
