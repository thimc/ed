package main

import (
	"bufio"
	"io"
	"unicode/utf8"
)

// EOF symbolizes the end of the file. It's value is set to the maximum
// value of a UTF-8 encoded rune, but the value itself has no meaning.
const EOF rune = utf8.UTFMax

// tokenizer is a buffered IO reader that implements peek functionality.
type tokenizer struct {
	tok    rune
	tokpos int
	*bufio.Reader
}

// newTokenizer creates a new Tokenizer and initializes the underlying
// `bufio.Reader` to `r`.
func newTokenizer(r io.Reader) *tokenizer {
	return &tokenizer{
		Reader: bufio.NewReader(r),
	}
}

// token reads (consumes) one rune from the input and returns it. On
// error it returns EOF as a rune.
func (t *tokenizer) token() rune {
	var err error
	t.tok, _, err = t.ReadRune()
	if err != nil {
		t.tok = EOF
		return t.tok
	}
	t.tokpos++
	return t.tok
}

// peek peeks at the next token rune without consuming it. On error it
// returns EOF as a rune.
func (t *tokenizer) peek() rune {
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
