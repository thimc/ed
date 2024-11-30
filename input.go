package main

import (
	"bufio"
	"strings"
	"unicode/utf8"
)

const EOF rune = -1

type input struct {
	*bufio.Scanner
	buf string
	pos int
}

func (i *input) match(s string) bool { return strings.ContainsAny(string(i.token()), s) }

func (i *input) doInput(s string) { i.buf, i.pos = s, 0 }

func (i *input) eof() bool { return i.pos >= len(i.buf) }

func (i *input) consume() {
	if i.eof() {
		return
	}
	_, n := utf8.DecodeRuneInString(i.buf[i.pos:])
	i.pos += n
}

func (i *input) token() rune {
	if i.eof() {
		return EOF
	}
	tok, _ := utf8.DecodeRuneInString(i.buf[i.pos:])
	return tok
}

func (i *input) insert(r rune) {
	i.buf = string(r) + i.buf
	i.pos = max(i.pos-1, 0)
}

func (i *input) Scan() bool {
	eof := i.Scanner.Scan()
	i.doInput(i.Scanner.Text())
	return eof
}
