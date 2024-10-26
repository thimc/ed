package main

import (
	"bufio"
	"io"
	"unicode/utf8"
)

const EOF rune = -1

// tokenizer is a buffered IO reader that implements peek functionality.
type tokenizer struct {
	tok    rune
	tokpos int
	*bufio.Reader
}

// newTokenizer creates a new Tokenizer.
func newTokenizer(r io.Reader) *tokenizer {
	return &tokenizer{
		Reader: bufio.NewReader(r),
	}
}

// consume consumes one rune from the input and returns it.
func (t *tokenizer) consume() rune {
	var err error
	t.tok, _, err = t.ReadRune()
	if err != nil {
		t.tok = EOF
		return t.tok
	}
	t.tokpos++
	return t.tok
}

// peek peeks at the next token rune without consuming it.
func (t *tokenizer) peek() rune {
	if t.Buffered() < 1 || t.tok == EOF {
		return EOF
	}
	for n := utf8.UTFMax; n > 0; n-- {
		b, err := t.Peek(n)
		if err != nil {
			continue
		}
		r, _ := utf8.DecodeRune(b)
		return r
	}
	return EOF
}
