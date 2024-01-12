package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
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
	Path        string          // file path
	Dirty       bool            // modified
	Lines       []string        // File buffer
	mark        [25]int         // a to z
	Dot         int             // current position
	Start       int             // start position
	End         int             // end position
	input       []byte          // user input
	addrCount   int             // number of addresses in the current input
	addr        int             // internal address
	s           scanner.Scanner // token scanner for the input byte array
	tok         rune            // current token
	Error       error           // previous error
	scroll      int             // previous scroll value
	search      string          // previous search criteria for /, ? or s
	replacestr  string          // previous s replacement
	Prompt      rune            // user prompt
	shellCmd    string          // previous command for !
	globalCmd   string          // previous command used by g, G, v and V
	printErrors bool            // toggle errors
	Silent      bool            // chatty
	sigch       chan os.Signal  // signals caught by ed
	sigint      bool            // if sigint was caught
	in          io.Reader       // standard input
	out         io.Writer       // standard output
	err         io.Writer       // standard error
}

// NewEditor returns a new Editor.
func NewEditor(stdin io.Reader, stdout io.Writer, stderr io.Writer) *Editor {
	ed := Editor{
		Lines: []string{},
		sigch: make(chan os.Signal, 1),
		in:    stdin,
		out:   stdout,
		err:   stderr,
	}
	ed.setupSignals()
	return &ed
}

// ReadInput reads user input from the io.Reader until it encounters
// a newline symbol (\n') or EOF. After that it sets up the scanner
// and tokenizer.
func (ed *Editor) ReadInput(r io.Reader) error {
	ed.input = []byte{}
	buf := make([]byte, 1)
	if ed.Prompt != 0 {
		fmt.Fprintf(ed.err, "%c", ed.Prompt)
	}
	for {
		n, err := r.Read(buf)
		if n == 0 {
			if len(ed.input) == 0 {
				return errors.New("EOF")
			}
			break
		}
		if buf[0] == '\n' {
			break
		}
		ed.input = append(ed.input, buf[0])
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
				ed.WriteFile(1, len(ed.Lines), defaultHangupFile)
			}
		case syscall.SIGINT:
			fmt.Fprintf(ed.err, "%s\n", ErrDefault)
			ed.sigint = true
		}
	}()
}

// ReadFile function will open the file specified by 'path,' read its
// input into the internal file buffer, and set the cursor position
// (dot) to the last line of the buffer. If no errors occur, the size
// of the file in bytes will be printed to the err io.Writer.
func (ed *Editor) ReadFile(path string) ([]string, error) {
	var lines []string
	file, err := os.Open(path)
	if err != nil {
		return lines, ErrCannotOpenFile
	}
	defer file.Close()
	stat, err := os.Stat(path)
	if err != nil {
		return lines, ErrCannotReadFile
	}
	s := bufio.NewScanner(file)
	for s.Scan() {
		lines = append(lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return lines, err
	}
	ed.Path = path
	ed.End = len(lines)
	ed.Start = ed.End
	ed.Dot = ed.End
	if !ed.Silent {
		fmt.Fprintf(ed.err, "%d\n", stat.Size())
	}
	return lines, nil
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
	for i := start - 1; i < end; i++ {
		var line string = ed.Lines[i]
		_, err := file.WriteString(line + "\n")
		if err != nil {
			return err
		}
		siz += len(line) + 1
	}
	ed.Dirty = false
	if !ed.Silent {
		fmt.Fprintf(ed.err, "%d\n", siz)
	}
	return err
}

// AppendFile will open the file 'path' and append the lines starting
// at index 'start' until 'end.' If successful, the current buffer
// will no longer be considered dirty.
func (ed *Editor) AppendFile(start, end int, path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
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
	if !ed.Silent {
		fmt.Fprintf(ed.err, "%d\n", siz)
	}
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
	if ctok == ' ' {
		ctok = cs.Scan()
	}
	for ctok != scanner.EOF {
		parsed += string(ctok)
		if ctok != '\\' && cs.Peek() == '%' {
			if ed.Path == "" {
				return output, ErrNoFileName
			}
			ctok = cs.Scan()
			parsed += ed.Path
		}
		ctok = cs.Scan()
	}
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
	ed.shellCmd = command
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

// checkRange will verify that the Start and End positions are valid
// numbers and within the size of the buffer if the current command
// is expected to use these variables.
func (ed *Editor) checkRange() error {
	skipCmds := []rune{'q', 'Q', 'e', 'E', 'f', 'i', 'a', 'H', 'h', 'P', 'r', '!', '='}
	for _, cmd := range skipCmds {
		if ed.token() == cmd {
			return nil
		}
	}
	if ed.Start > ed.End || ed.Start < 1 || ed.End < 1 || ed.End > len(ed.Lines) {
		return ErrInvalidAddress
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
	return str
}

// scanNumber will advance the tokenizer while the current token is a
// digit and convert the parsed data to an integer.
func (ed *Editor) scanNumber() (int, error) {
	var n, start, end int
	var err error
	start = ed.s.Position.Offset
	for unicode.IsDigit(ed.token()) {
		ed.nextToken()
	}
	end = ed.s.Position.Offset
	num := string(ed.input[start:end])
	n, err = strconv.Atoi(num)
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

// dump is a helper function that is used to print the state of the program.
// The start, end and dot index values are printed to standard output.
// The internal address value and the address counter is also printed.
func (ed *Editor) dump() {
	fmt.Printf("start=%d | end=%d | dot=%d | addr=%d | addrcount=%d | ",
		ed.Start, ed.End, ed.Dot, ed.addr, ed.addrCount)
	fmt.Printf("offset=%d | eof=%t | token='%c' | ",
		ed.s.Pos().Offset, ed.token() == scanner.EOF, ed.token())
	fmt.Printf("buffer_len=%d\n", len(ed.Lines))
}
