package main

import (
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func (ed *Editor) parse() error {
	var addr int
	ed.addrc = 0
	ed.first, ed.second = ed.dot, ed.dot
	defer func() {
		ed.addrc = min(ed.addrc, 2)
		if ed.addrc == 1 || ed.addrc > 0 && ed.second != addr {
			ed.first = ed.second
		}
	}()
	for {
		var err error
		addr, err = ed.nextAddress()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		ed.addrc += 1
		if addr < 1 {
			ed.second = 0
			break
		}
		ed.first = ed.second
		ed.second = addr
		if !ed.match(",;") {
			break
		}
		r := ed.token()
		ed.consume()
		if r == ';' {
			ed.dot = addr
		}
	}
	return nil
}

func (ed *Editor) nextAddress() (int, error) {
	var (
		addr  = ed.dot
		i     = ed.input.pos
		err   error
		first = true
	)
	ed.skipWhitespace()
	for ; ; first = false {
		switch {
		case unicode.IsDigit(ed.token()), ed.match("+-^"):
			r := ed.token()
			if !unicode.IsDigit(r) {
				ed.consume()
			}
			var n int
			ed.skipWhitespace()
			if unicode.IsDigit(ed.token()) {
				n, err = ed.scanNumber()
				if err != nil {
					return -1, err
				}
			} else if !unicode.IsSpace(r) {
				n = 1
			}
			switch r {
			case '-', '^':
				addr -= n
			case '+':
				addr += n
			default:
				addr = n
			}
		case ed.match(".$"):
			if !first {
				return -1, ErrInvalidAddress
			}
			addr = len(ed.lines)
			if ed.token() == '.' {
				addr = ed.dot
			}
			ed.consume()
		case ed.match("?/"):
			if !first {
				return -1, ErrInvalidAddress
			}
			if len(ed.lines) < 1 {
				return -1, ErrNoMatch
			}
			r := ed.token()
			ed.consume()
			search, eof := ed.scanStringUntil(r)
			if !eof && ed.token() == r {
				ed.consume()
			}
			var re *regexp.Regexp
			if search == "" {
				if ed.re == nil {
					return -1, ErrNoPrevPattern
				}
				re = ed.re
			} else {
				re, err = regexp.Compile(search)
				if err != nil {
					return -1, err
				}
				ed.re = re
			}
			i := ed.dot
			if r != '/' {
				i--
			}
			var found bool
			for {
				if r == '/' {
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
				} else if i+1 == ed.dot {
					break
				}
			}
			if !found {
				return -1, ErrNoMatch
			}
		case ed.token() == '\'':
			if !first {
				return -1, ErrInvalidAddress
			}
			ed.consume()
			r := ed.token()
			m := int(r) - 'a'
			if m < 0 || m >= len(ed.mark) || !unicode.IsLower(r) {
				return -1, ErrInvalidMark
			}
			maddr := ed.mark[m]
			if maddr < 1 || maddr > len(ed.lines) {
				return -1, ErrInvalidAddress
			}
			addr = maddr
			ed.consume()
		case ed.match("%,;"):
			if first {
				ed.addrc += 1
				ed.second = 1
				if ed.token() == ';' {
					ed.second = ed.dot
				}
				ed.consume()
				if addr, err = ed.nextAddress(); err != nil {
					addr = len(ed.lines)
				}
			}
			fallthrough
		default:
			if ed.input.pos == i {
				return -1, io.EOF
			}
			if addr < 0 || addr > len(ed.lines) {
				ed.addrc += 1
				return -1, ErrInvalidAddress
			}
			return addr, nil
		}
	}
}

func (ed *Editor) skipWhitespace() {
	for ed.match(" \t") {
		ed.consume()
	}
}

func (ed *Editor) scanNumber() (int, error) {
	var sb strings.Builder
	for unicode.IsDigit(ed.token()) {
		sb.WriteRune(ed.token())
		ed.consume()
	}
	n, err := strconv.Atoi(sb.String())
	if err != nil {
		return -1, ErrNumberOutOfRange
	}
	return n, nil
}

func (ed *Editor) scanString() string {
	s, _ := ed.scanStringUntil('\n')
	return s
}

func (ed *Editor) scanStringUntil(delim rune) (str string, eof bool) {
	var sb strings.Builder
	for ed.token() != EOF && ed.token() != delim {
		sb.WriteRune(ed.token())
		ed.consume()
	}
	return sb.String(), ed.token() == EOF
}
