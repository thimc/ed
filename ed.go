package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/scanner"
)

const (
	defaultPrompt     = '*'
	defaultHangupFile = "ed.hup"
)

var (
	ErrDefault             = errors.New("?") // descriptive error message, don't you think?
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidMark         = errors.New("invalid mark character")
	ErrInvalidNumber       = errors.New("number out of range")
	ErrCannotOpenFile      = errors.New("cannot open input file")
	ErrNoFileName          = errors.New("no current filename")
	ErrUnknownCmd          = errors.New("unknown command")
	ErrUnexpectedCmdSuffix = errors.New("unexpected command suffix")
	ErrInvalidCmdSuffix    = errors.New("invalid command suffix")
	ErrDestinationExpected = errors.New("destination expected")
	ErrFileModified        = errors.New("warning: file modified")
	ErrNoPrevPattern       = errors.New("no previous pattern")
	ErrNoMatch             = errors.New("no match")
	ErrNoCmd               = errors.New("no command")
	ErrZero                = errors.New("0")
)

type Editor struct {
	Path  string
	Dirty bool
	Lines []string
	Mark  ['z' - 'a']int

	Dot   int
	Start int
	End   int

	input     []byte
	addrcount int
	addr      int
	s         scanner.Scanner
	tok       rune

	Search string
	Error  error
	Prompt rune
	Cmd    string

	printErrors bool

	sigch  chan os.Signal
	sigint bool

	in  io.Reader
	out io.Writer
	err io.Writer
}

// NewEditor returns a new Editor.
func NewEditor(stdin io.Reader, stdout io.Writer, stderr io.Writer) *Editor {
	ed := Editor{
		Lines:  []string{},
		Prompt: defaultPrompt,
		sigch:  make(chan os.Signal, 1),
		in:     stdin,
		out:    stdout,
		err:    stderr,
	}
	ed.setupSignals()
	return &ed
}

// ReadInput() reads until the first new line '\n' or EOF
// and configures the internal scanner and tokenizer for ed.
func (ed *Editor) ReadInput(r io.Reader) error {
	ed.input = []byte{}
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n == 0 {
			break
		}
		ed.input = append(ed.input, buf[0])
		if buf[0] == '\n' {
			break
		}
		// FIXME: Check the error _after_ the value is
		// appended to get a "EOF" symbol into the array.
		if err != nil {
			return err
		}
	}
	ed.setupScanner()
	return nil
}

// setupScanner sets up a token scanner and initializes the
// internal token variable to that of the buffer.
func (ed *Editor) setupScanner() {
	ed.s.Init(bytes.NewReader(ed.input))
	ed.s.Mode = scanner.ScanStrings
	ed.s.Whitespace ^= scanner.GoWhitespace
	ed.nextToken()
}

// setupSignals sets up signal handlers for SIGHUP and SIGINT.
func (ed *Editor) setupSignals() {
	ed.sigint = false
	signal.Notify(ed.sigch, syscall.SIGHUP, syscall.SIGINT)
	go func() {
		sig := <-ed.sigch
		switch sig {
		case syscall.SIGHUP:
			if ed.Dirty {
				log.Printf("Received SIGHUP and file is dirty\n")
				ed.WriteFile(1, len(ed.Lines), defaultHangupFile)
			}
		case syscall.SIGINT:
			log.Printf("Received SIGINT\n")
			fmt.Fprintf(os.Stderr, "%s\n", ErrDefault)
			ed.sigint = true
		}
	}()
}

// debug function used to print the "stack frame" of the application,
// the start, end and dot index values are printed to standard output.
// The internal address value and the address counter is also printed.
func (ed *Editor) dump() {
	log.Printf("start=%d | end=%d | dot=%d | addr=%d | addrcount=%d | ",
		ed.Start, ed.End, ed.Dot, ed.addr, ed.addrcount)
	log.Printf("offset=%d | eof=%t | token='%c' | ",
		ed.s.Pos().Offset, ed.token() == scanner.EOF, ed.token())
	log.Printf("buffer_len=%d\n", len(ed.Lines))
}

// checkRange checks if the Start and End values are valid numbers
// and within the size of the current buffer.
func (ed *Editor) checkRange() error {
	if len(ed.Lines) > 0 {
		if ed.Start > ed.End || ed.Start < 1 || ed.End < 1 || ed.End > len(ed.Lines) {
			return ErrInvalidAddress
		}
	}
	return nil
}
