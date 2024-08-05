package main

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"unicode"
)

// parse parses the user input for valid addresses and returns if the
// current token is not a valid range command or EOF.
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
	if ed.addrc > 2 {
		ed.addrc = 2
	}
	if ed.addrc == 1 || ed.end != addr {
		ed.start = ed.end
	}

	return nil
}

// nextAddress extracts the next address in the user input.
func (ed *Editor) nextAddress() (int, error) {
	var (
		addr  = ed.dot
		err   error
		first = true
	)
	ed.skipWhitespace()
	startpos := ed.tokpos
	for starttok := ed.tok; ; first = false {
		switch {
		case unicode.IsDigit(ed.tok) || ed.tok == '+' || ed.tok == '-' || ed.tok == '^':
			mod := ed.tok
			if !unicode.IsDigit(mod) {
				ed.token()
			}
			var n int
			ed.skipWhitespace()
			if unicode.IsDigit(ed.tok) {
				n, err = ed.scanNumber()
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
			addr = len(ed.lines)
			if ed.tok == '.' {
				addr = ed.dot
			}
			ed.token()
		case ed.tok == '?', ed.tok == '/':
			if !first {
				return -1, ErrInvalidAddress
			}
			var mod = ed.tok
			ed.token()
			var search = ed.scanStringUntil(mod)
			re, err := regexp.Compile(search)
			if err != nil {
				return -1, err
			}
			if search == "" {
				if ed.re == nil {
					return -1, ErrNoPrevPattern
				}
				re = ed.re
			}
			ed.re = re
			var (
				found bool
				i     = ed.dot
			)
			if len(ed.lines) < 1 {
				return -1, ErrNoMatch
			}
			if mod != '/' {
				i--
			}
			for {
				if mod == '/' {
					i++
				} else {
					i--
					if i < 0 {
						i = len(ed.lines) - 1
					}
				}
				i %= len(ed.lines)
				if re.MatchString(ed.lines[i]) {
					addr = i + 1
					found = true
					break
				}
				if i == ed.dot {
					break
				}
			}
			if !found {
				return -1, ErrNoMatch
			}
		case ed.tok == '\'':
			if !first {
				return -1, ErrInvalidAddress
			}
			var r = ed.token()
			if r == EOF || !unicode.IsLower(r) {
				return -1, ErrInvalidMark
			}
			var mark = int(r) - 'a'
			if mark < 0 || mark >= len(ed.mark) {
				return -1, ErrInvalidMark
			}
			var maddr = ed.mark[mark]
			if maddr < 1 || maddr > len(ed.lines) {
				return -1, ErrInvalidAddress
			}
			addr = maddr
			ed.token()
		case ed.tok == '%' || ed.tok == ',' || ed.tok == ';':
			if first {
				ed.addrc++
				ed.end = 1
				if ed.tok == ';' {
					ed.end = ed.dot
				}
				ed.token()
				if addr, err = ed.nextAddress(); err != nil {
					addr = len(ed.lines)
				}
			}
			fallthrough
		default:
			if ed.tok == starttok {
				return -1, io.EOF
			}
			if addr < 0 || addr > len(ed.lines) {
				ed.addrc++
				return -1, ErrInvalidAddress
			}
			return addr, nil
		}
	}
}

// check validates if n, m are valid depending on how many addresses
// were previously parsed. check returns error "invalid address" if
// the positions are out of bounds.
func (ed *Editor) check(n, m int) error {
	if ed.addrc == 0 {
		ed.start = n
		ed.end = m
	}
	if ed.start > ed.end || ed.start < 1 || ed.end > len(ed.lines) {
		return ErrInvalidAddress
	}
	return nil
}

// scanNumber scans the user input for a number and will advance until
// the current token is not a valid digit.
func (ed *Editor) scanNumber() (int, error) {
	var s string
	for unicode.IsDigit(ed.tok) {
		s += string(ed.tok)
		ed.token()
	}
	return strconv.Atoi(s)
}

// scanString scans the user input until EOF or new line.
func (ed *Editor) scanString() string {
	var str string
	for ed.tok != EOF && ed.tok != '\n' {
		str += string(ed.tok)
		ed.token()
	}
	return str
}

// scanStringUntil works like `scanString` but will continue until it
// sees (and consumes) `delim`. If `delim` is not found it continues
// until EOF.
func (ed *Editor) scanStringUntil(delim rune) string {
	var str string
	for ed.tok != EOF && ed.tok != '\n' && ed.tok != delim {
		str += string(ed.tok)
		ed.token()
	}
	if ed.tok == delim {
		ed.token()
	}
	return str
}

func (e *Editor) skipWhitespace() {
	for e.tok == ' ' || e.tok == '\t' {
		e.token()
	}
}
