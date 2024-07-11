package main

import (
	"regexp"
	"text/scanner"
	"unicode"
)

// nextAddress attempts to extract the next valid address.
func (ed *Editor) nextAddress() (int, error) {
	ed.addr = ed.dot
	var (
		mod   rune
		first bool
	)
	for first = true; ; first = false {
		for {
			switch ed.tok {
			case ' ', '\t', '\n', '\r':
				ed.tok = ed.s.Scan()
				continue
			}
			break
		}
		switch {
		case ed.tok == '.', ed.tok == '$':
			if !first {
				return 0, ErrInvalidAddress
			}
			ed.addr = len(ed.Lines)
			if ed.tok == '.' {
				ed.addr = ed.dot
			}
			ed.tok = ed.s.Scan()
		case ed.tok == '?', ed.tok == '/':
			if !first {
				return 0, ErrInvalidAddress
			}
			var mod = ed.tok
			ed.tok = ed.s.Scan()
			var search = ed.scanStringUntil(mod)
			if ed.tok == mod {
				ed.tok = ed.s.Scan()
			}
			if search == "" {
				search = ed.search
				if ed.search == "" {
					return 0, ErrNoPrevPattern
				}
			}
			ed.search = search
			var s, e = 0, len(ed.Lines)
			if mod == '?' {
				s = ed.start - 2
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
			if !first {
				return 0, ErrInvalidAddress
			}
			ed.tok = ed.s.Scan()
			var r = ed.tok
			ed.tok = ed.s.Scan()
			if r == scanner.EOF || !unicode.IsLower(r) {
				return 0, ErrInvalidMark
			}
			var mark = int(r) - 'a'
			if mark < 0 || mark > len(ed.mark) {
				return 0, ErrInvalidMark
			}
			var maddr = ed.mark[mark]
			if maddr <= 0 || maddr > len(ed.Lines) {
				return 0, ErrInvalidAddress
			}
			ed.addr = maddr
		case ed.tok == '+', ed.tok == '-', ed.tok == '^':
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
		case ed.tok == ';', ed.tok == '%', ed.tok == ',':
			var r = ed.tok
			if first {
				ed.tok = ed.s.Scan()
				n, err := ed.nextAddress()
				if err != nil {
					return 0, err
				}
				ed.addr = n
				if n == ed.dot && ed.start == ed.end {
					ed.start = 1
					if r == ';' {
						ed.start = ed.end
					}
					ed.dot = len(ed.Lines)
					ed.addr = ed.dot
					ed.end = ed.dot
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

// parseRange parses user input to extract the specified line or range
// on which the user intends to execute commands.
func (ed *Editor) parseRange() error {
	var (
		n   int
		err error
	)
	ed.addrCount = 0
	ed.start = ed.dot
	ed.end = ed.dot
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
		ed.start = ed.end
		ed.end = ed.addr
		if ed.tok != ',' && ed.tok != ';' {
			break
		} else if ed.s.Peek() == ';' {
			ed.dot = ed.addr
		}
		if ed.tok == scanner.EOF {
			break
		}
	}
end:
	if ed.addrCount == 1 || ed.end != ed.addr {
		ed.start = ed.end
	}
	ed.dot = ed.end

	skipCmds := []rune{'a', 'e', 'E', 'f', 'h', 'H', 'i', 'P', 'q', 'Q', 'r', 'u', '!', '='}
	for _, cmd := range skipCmds {
		if ed.tok == cmd {
			return nil
		}
	}
	if ed.start > ed.end || ed.start < 1 || ed.end < 1 || ed.end > len(ed.Lines) || ed.addr > len(ed.Lines) {
		return ErrInvalidAddress
	}

	return nil
}
