package main

import (
	"errors"
	"io"
	"log"
	"strconv"
	"unicode"
)

// scanString scans the user input until EOF. New lines and carriage
// return symbols are ignored.
func (ed *Editor) scanString() string {
	var str string
	for ed.tok != EOF && ed.tok != '\n' {
		str += string(ed.tok)
		ed.token()
	}
	return str
}

// scanStringUntil works like `scanString` but will continue until it
// sees `delim` or EOF.
func (ed *Editor) scanStringUntil(delim rune) string {
	var str string
	for ed.tok != EOF && ed.tok != delim {
		if ed.tok != '\n' && ed.tok != '\r' {
			str += string(ed.tok)
		}
		ed.token()
	}
	return str
}

func (e *Editor) parseNumber() (int, error) {
	var s string
	for unicode.IsDigit(e.tok) {
		s += string(e.tok)
		e.token()
	}
	return strconv.Atoi(s)
}

func (e *Editor) skipWhitespace() {
	for e.tok == ' ' || e.tok == '\t' {
		e.tok = e.token()
	}
}

func (ed *Editor) nextAddress() (int, error) {
	var (
		addr  = ed.dot
		err   error
		first = true
	)
	ed.skipWhitespace()
	for starttok := ed.tok; ; first = false {
		startpos := ed.tokpos

		switch {
		case unicode.IsDigit(ed.tok) || ed.tok == '+' || ed.tok == '-' || ed.tok == '^':
			mod := ed.tok
			if !unicode.IsDigit(mod) {
				ed.token()
			}
			var n int
			ed.skipWhitespace()
			if unicode.IsDigit(ed.tok) {
				n, err = ed.parseNumber()
				if err != nil {
					return -1, err
				}
			} else if !unicode.IsSpace(mod) {
				n = 1
			}
			switch mod {
			case '-', '^':
				addr -= n
			case '+':
				addr += n
			default:
				addr = n
			}
		case ed.tok == '.' || ed.tok == '$':
			if ed.tokpos != startpos {
				return -1, ErrInvalidAddress
			}
			addr = len(ed.Lines)
			if ed.tok == '.' {
				addr = ed.dot
			}
			ed.token()
		case ed.tok == '?', ed.tok == '/':
			// if !first {
			// 	return 0, ErrInvalidAddress
			// }
			// var mod = e.tok
			// e.token()
			// var search = e.scanStringUntil(mod)
			// if e.tok == mod {
			// 	e.token()
			// }
			// if search == "" {
			// 	search = e.search
			// 	if e.search == "" {
			// 		return 0, ErrNoPrevPattern
			// 	}
			// }
			// e.search = search
			// var s, e = 0, len(e.Lines)
			// if mod == '?' {
			// 	s = e.start - 2
			// 	e = 0
			// }
			// for i := s; i != e; {
			// 	if i < 0 || i > len(e.Lines) {
			// 		return 0, ErrNoMatch
			// 	}
			// 	match, err := regexp.MatchString(search, e.Lines[i])
			// 	if err != nil {
			// 		return 0, ErrNoMatch
			// 	}
			// 	if match {
			// 		addr = i + 1
			// 		return addr, nil
			// 	}
			// 	if mod == '/' {
			// 		i++
			// 	} else {
			// 		i--
			// 	}
			// }
			// return 0, ErrNoMatch
		case ed.tok == '\'':
			// if !first {
			// 	return 0, ErrInvalidAddress
			// }
			// e.tok = e.s.Scan()
			// var r = e.tok
			// e.tok = e.s.Scan()
			// if r == scanner.EOF || !unicode.IsLower(r) {
			// 	return 0, ErrInvalidMark
			// }
			// var mark = int(r) - 'a'
			// if mark < 0 || mark > len(e.mark) {
			// 	return 0, ErrInvalidMark
			// }
			// var maddr = e.mark[mark]
			// if maddr <= 0 || maddr > len(e.Lines) {
			// 	return 0, ErrInvalidAddress
			// }
			// addr = maddr
		case ed.tok == '%' || ed.tok == ',' || ed.tok == ';':
			if first {
				ed.addrc++
				ed.end = 1
				if ed.tok == ';' {
					ed.end = ed.dot
				}
				ed.token()
				if addr, err = ed.nextAddress(); err != nil {
					addr = len(ed.Lines)
				}
			}
			fallthrough
		default:
			if ed.tok == starttok {
				return -1, io.EOF
			}
			if addr < 1 || addr > len(ed.Lines) {
				log.Printf("addr(%d) < 1 || addr > %d", addr, len(ed.Lines))
				return -1, ErrInvalidAddress
			}
			return addr, nil
		}
	}
}

func (ed *Editor) parse() error {
	var (
		addr int
		err  error
	)
	ed.addrc = 0
	ed.start = ed.dot
	ed.end = ed.dot
	for {
		addr, err = ed.nextAddress()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if addr < 1 {
			break
		}
		ed.addrc++
		ed.start = ed.end
		ed.end = addr
		if ed.tok != ',' && ed.tok != ';' {
			break
		} else if ed.token() == ';' {
			ed.dot = addr
		}
	}
	if ed.addrc = min(ed.addrc, 2); ed.addrc == 1 || ed.end != addr {
		ed.start = ed.end
	}
	return nil
}

func (ed *Editor) check(n, m int) error {
	if ed.addrc == 0 {
		ed.start = n
		ed.end = m
	}
	if ed.start > ed.end || ed.start < 1 || ed.end > len(ed.Lines) {
		return ErrInvalidAddress
	}
	return nil
}
