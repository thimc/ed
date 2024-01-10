package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"text/scanner"
	"unicode"
)

const (
	defaultPrompt     = '*'
	defaultHangupFile = "ed.hup"
)

var (
	ErrDefault             = errors.New("?") // descriptive error message, don't you think?
	ErrCannotOpenFile      = errors.New("cannot open input file")
	ErrCannotReadFile      = errors.New("cannot read input file")
	ErrDestinationExpected = errors.New("destination expected")
	ErrFileModified        = errors.New("warning: file modified")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidCmdSuffix    = errors.New("invalid command suffix")
	ErrInvalidDestination  = errors.New("invalid destination")
	ErrInvalidMark         = errors.New("invalid mark character")
	ErrInvalidNumber       = errors.New("number out of range")
	ErrInvalidPatternDelim = errors.New("invalid pattern delimiter")
	ErrNoCmd               = errors.New("no command")
	ErrNoFileName          = errors.New("no current filename")
	ErrNoMatch             = errors.New("no match")
	ErrNoPrevPattern       = errors.New("no previous pattern")
	ErrNoPreviousSub       = errors.New("no previous substitution")
	ErrUnexpectedCmdSuffix = errors.New("unexpected command suffix")
	ErrUnknownCmd          = errors.New("unknown command")
	ErrZero                = errors.New("0")
)

type Editor struct {
	Path  string
	Dirty bool
	Lines []string
	Mark  ['z' - 'a']int // [25]int

	Dot   int
	Start int
	End   int

	input     []byte
	addrcount int
	addr      int
	s         scanner.Scanner
	tok       rune

	Search         string
	Error          error
	Prompt         rune
	Cmd            string
	globalCmd      string
	prevSubSearch  string
	prevSubReplace string

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

// ReadInput() reads from the in io.Reader until it encounters a newline
// symbol (\n') or EOF. After that it sets up the scanner and tokenizer.
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

// ReadFile function will open the file specified by 'path,' read its
// input into the internal file buffer, and set the cursor position
// (dot) to the last line of the buffer. If no errors occur, the size
// of the file in bytes will be printed to the err io.Writer.
func (ed *Editor) ReadFile(path string) (int64, []string, error) {
	var lines []string
	var siz int64
	file, err := os.Open(path)
	if err != nil {
		return siz, lines, ErrCannotOpenFile
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return siz, lines, ErrCannotOpenFile
	}
	siz = stat.Size()
	s := bufio.NewScanner(file)
	for s.Scan() {
		ed.Lines = append(ed.Lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return siz, lines, err
	}
	ed.Path = path
	ed.Dot = len(ed.Lines)
	ed.Start = ed.Dot
	ed.End = ed.Dot
	ed.addr = -1
	return siz, lines, nil
}

// WriteFile function will attempt to write the lines from index 'start'
// to 'end' in the file specified by 'path.' If successful, the current
// buffer will no longer be considered dirty.
func (ed *Editor) WriteFile(start, end int, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var siz int
	log.Printf("Write range %d to %d to %s\n", start, end, path)
	for i := start - 1; i < end; i++ {
		var line string = ed.Lines[i]
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
		siz += len(line) + 1
	}
	ed.Dirty = false
	fmt.Fprintf(ed.err, "%d\n", siz)
	return err
}

// AppendFile will open the file 'path' and append the lines starting
// at index 'start' until 'end.' If successful, the current buffer
// will no longer be considered dirty.
func (ed *Editor) AppendFile(start, end int, path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	defer file.Close()
	log.Printf("Append range %d to %d to %s\n", start, end, path)
	var siz int
	for i := start - 1; i < end; i++ {
		var line string = ed.Lines[i]
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
		siz += len(line) + 1
	}
	ed.Dirty = false
	fmt.Fprintf(ed.err, "%d\n", siz)
	return err
}

// Shell runs the 'command' in /bin/sh and returns the standard output.
// It will replace any unescaped '%' with the name of the current buffer.
func (ed *Editor) Shell(command string) ([]string, error) {
	var output []string
	var cs scanner.Scanner
	cs.Init(strings.NewReader(command))
	cs.Mode = scanner.ScanChars
	cs.Whitespace ^= scanner.GoWhitespace
	var parsed string
	var ctok rune = cs.Scan()
	for ctok != scanner.EOF {
		parsed += string(ctok)
		if ctok != '\\' && cs.Peek() == '%' {
			ctok = cs.Scan()
			log.Printf("Replacing %% with '%s'\n", ed.Path)
			parsed += ed.Path
		}
		ctok = cs.Scan()
	}
	log.Printf("Shell (parsed): '%s'\n", parsed)
	cmd := exec.Command("/bin/sh", "-c", parsed)
	stdout, err := cmd.StdoutPipe()
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		return output, err
	}
	if err != nil {
		return output, err
	}
	s := bufio.NewScanner(stdout)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		output = append(output, s.Text())
	}
	if err := cmd.Wait(); err != nil {
		return output, err
	}
	if err := s.Err(); err != nil {
		return output, err
	}
	ed.Cmd = command
	return output, err
}

// ReadInsert will read from the internal in io.Reader until it
// encounters a newline (\n) or is interrupted by SIGINT.
func (ed *Editor) ReadInsert() (string, error) {
	var buf bytes.Buffer
	var b []byte = make([]byte, 1)
	for {
		if ed.sigint {
			return "", fmt.Errorf("Canceled by SIGINT")
		}
		if _, err := ed.in.Read(b); err != nil {
			return buf.String(), err
		}
		if b[0] == '\n' {
			break
		}
		if err := buf.WriteByte(b[0]); err != nil {
			return buf.String(), err
		}
	}
	return buf.String(), nil
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

// scanString will advance the tokenizer, scanning the input buffer
// until it reaches EOF, and return the collected tokens as a string.
// Newlines (\n) and carriage returns (\r) are ignored.
func (ed *Editor) scanString() string {
	var str string
	for ed.token() != scanner.EOF {
		if ed.token() != '\n' && ed.token() != '\r' {
			str += string(ed.token())
		}
		ed.nextToken()
	}
	log.Printf("scanString(): '%s'\n", str)
	return str
}

// scanStringUntil will advance the tokenizer, scanning the input
// buffer until it reaches the delimiter 'delim' or EOF, and return
// the collected tokens as a string.  Newlines (\n) and carriage returns
// (\r) are ignored.
func (ed *Editor) scanStringUntil(delim rune) string {
	var str string
	for ed.token() != scanner.EOF && ed.token() != delim {
		if ed.token() != '\n' && ed.token() != '\r' {
			str += string(ed.token())
		}
		ed.nextToken()
	}
	log.Printf("scanStringUntil(): '%s'\n", str)
	return str
}

// scanNumber will advance the tokenizer while the current token is a
// digit and convert the parsed data to an integer
func (ed *Editor) scanNumber() (int, error) {
	var n, start, end int
	var err error

	start = ed.s.Position.Offset
	for unicode.IsDigit(ed.token()) {
		ed.nextToken()
	}
	end = ed.s.Position.Offset

	num := string(ed.input[start:end])
	log.Printf("Convert num: '%s'\n", num)
	n, err = strconv.Atoi(num)
	log.Printf("ConsumeNumber(): '%d' err=%t\n", n, err != nil)
	return n, err
}

// skipWhitespace advances the tokenizer until the current token is
// not a white space, tab indent, or a newline.
func (ed *Editor) skipWhitespace() {
	for ed.token() == ' ' || ed.token() == '\t' || ed.token() == '\n' {
		ed.nextToken()
	}
}

// token returns the current token.
func (ed *Editor) token() rune {
	return ed.tok
}

// nextToken will advance the tokenizer and set the current token
func (ed *Editor) nextToken() {
	ed.tok = ed.s.Scan()
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
