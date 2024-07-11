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
	DefaultPrompt     = "*"
	DefaultHangupFile = "ed.hup"
	DefaultShell      = "/bin/sh"
)

type undoType int

const (
	undoAdd undoType = iota
	undoDelete
)

type undoAction struct {
	typ   undoType
	start int
	end   int
	d     int
	lines []string
}

var (
	ErrDefault             = errors.New("?") // descriptive error message, don't you think?
	ErrCannotNestGlobal    = errors.New("cannot nest global commands")
	ErrCannotOpenFile      = errors.New("cannot open input file")
	ErrCannotReadFile      = errors.New("cannot read input file")
	ErrDestinationExpected = errors.New("destination expected")
	ErrFileModified        = errors.New("warning: file modified")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidCmdSuffix    = errors.New("invalid command suffix")
	ErrInvalidDestination  = errors.New("invalid destination")
	ErrInvalidMark         = errors.New("invalid mark character")
	ErrInvalidNumber       = errors.New("number out of range")
	ErrInvalidRedirection  = errors.New("invalid redirection")
	ErrInvalidPatternDelim = errors.New("invalid pattern delimiter")
	ErrNoCmd               = errors.New("no command")
	ErrNoFileName          = errors.New("no current filename")
	ErrNoMatch             = errors.New("no match")
	ErrNoPrevPattern       = errors.New("no previous pattern")
	ErrNoPreviousSub       = errors.New("no previous substitution")
	ErrNothingToUndo       = errors.New("nothing to undo")
	ErrUnexpectedAddress   = errors.New("unexpected address")
	ErrUnexpectedCmdSuffix = errors.New("unexpected command suffix")
	ErrUnknownCmd          = errors.New("unknown command")
	ErrZero                = errors.New("0")
)

type Editor struct {
	path        string          // file path
	dirty       bool            // modified
	Lines       []string        // File buffer
	mark        [25]int         // a to z
	dot         int             // current position
	start       int             // start position
	end         int             // end position
	input       []byte          // user input
	addrCount   int             // number of addresses in the current input
	addr        int             // internal address
	s           scanner.Scanner // token scanner for the input byte array
	tok         rune            // current token
	undohist    [][]undoAction  // undo history
	globalUndo  []undoAction    // undo actions caught during global cmds
	g           bool            // global command state
	error       error           // previous error
	scroll      int             // previous scroll value
	search      string          // previous search criteria for /, ? or s
	replacestr  string          // previous s replacement
	showPrompt  bool            // toggle for displaying the prompt
	prompt      string          // user prompt
	shellCmd    string          // previous command for !
	globalCmd   string          // previous command used by g, G, v and V
	printErrors bool            // toggle errors
	silent      bool            // chatty
	sigch       chan os.Signal  // signals caught by ed
	sigint      bool            // if sigint was caught
	in          io.Reader       // standard input
	out         io.Writer       // standard output
	err         io.Writer       // standard error
}

// New creates a new instance of the Ed editor. It defaults to reading
// user input from the operating systems standard input, printing to
// standard output and errors to standard error. Signal handlers are
// created.
func New(opts ...OptionFunc) *Editor {
	ed := &Editor{
		sigch: make(chan os.Signal, 1),
		in:    os.Stdin,
		out:   os.Stdout,
		err:   os.Stderr,
	}
	for _, opt := range opts {
		opt(ed)
	}
	ed.setupSignals()
	return ed
}

type OptionFunc func(*Editor)

// WithStdin overrides the io.Reader from which ed will read user input from.
func WithStdin(r io.Reader) OptionFunc {
	return func(ed *Editor) {
		ed.in = r
	}
}

// WithStdout overrides the io.Reader from which ed will print output to.
func WithStdout(w io.Writer) OptionFunc {
	return func(ed *Editor) {
		ed.out = w
	}
}

// WithStderr overrides the io.Reader from which ed will print error messages to.
func WithStderr(w io.Writer) OptionFunc {
	return func(ed *Editor) {
		ed.err = w
	}
}

// WithPrompt overrides the user prompt that can be activated with the
// `P` command.
func WithPrompt(s string) OptionFunc {
	return func(ed *Editor) {
		ed.prompt = s
		ed.showPrompt = (ed.prompt != "")
	}
}

// WithFile sets the inital buffer of the editor. On error it ignores
// the state of the `H` command and prints the actual error to the
// editors stderr `io.Writer`.
func WithFile(path string) OptionFunc {
	return func(ed *Editor) {
		var err error
		ed.Lines, err = ed.readFile(path, true, true)
		if err != nil {
			fmt.Fprint(ed.err, err)
		}
	}
}

// WithSilent overrides the default silent mode of the editor. This mode
// is meant to be used with ed scripts. TODO: They are not fully
// supported yet.
func WithSilent(b bool) OptionFunc {
	return func(ed *Editor) {
		ed.silent = b
	}
}

// readInput reads user input from the io.Reader until it encounters
// a newline symbol (\n') or EOF. After that it sets up the scanner
// and tokenizer.
func (ed *Editor) readInput(r io.Reader) error {
	ed.input = []byte{}
	buf := make([]byte, 1)
	if ed.showPrompt {
		fmt.Fprintf(ed.err, "%s", ed.prompt)
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

	ed.s.Init(bytes.NewReader(ed.input))
	ed.s.Mode = scanner.ScanStrings
	ed.s.Whitespace ^= scanner.GoWhitespace
	ed.tok = ed.s.Scan()

	return nil
}

// setupSignals sets up signal handlers for SIGHUP and SIGINT.
func (ed *Editor) setupSignals() {
	ed.sigint = false
	signal.Notify(ed.sigch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		sig := <-ed.sigch
		switch sig {
		case syscall.SIGHUP:
			if ed.dirty {
				ed.writeFile(1, len(ed.Lines), DefaultHangupFile)
			}
		case syscall.SIGINT:
			fmt.Fprintln(ed.err, ErrDefault)
			ed.sigint = true
		case syscall.SIGQUIT:
			// ignore
		}
	}()
}

// readFile checks if the 'path' starts with a '!' and if so executes
// what it presumes to be a valid shell command in sh(1). If readFile
// deems 'path' not to be a shell expression it will attempt to open
// 'path' like a regular file.  If no error occurs and 'setdot' is
// true, the cursor positions are set to the last line of the buffer.
// If 'printsiz' is set to true, the size in bytes is printed to the
// 'err io.Writer'.
func (ed *Editor) readFile(path string, setdot bool, printsiz bool) ([]string, error) {
	var siz int64
	var cmd bool
	if len(path) > 0 {
		cmd = (path[0] == '!')
		if cmd {
			path = path[1:]
		}
	}
	var lines []string
	switch cmd {
	case true:
		if path == "" {
			path = ed.shellCmd
			if path == "" {
				return lines, ErrNoCmd
			}
		}
		output, err := ed.shell(path)
		if err != nil {
			return lines, ErrZero
		}
		for _, line := range output {
			lines = append(lines, line)
			siz += int64(len(line)) + 1
		}
	case false:
		ed.path = path
		if path == "" {
			path = ed.path
			if path == "" {
				return lines, ErrNoFileName
			}
		}
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
		siz = stat.Size()
	}
	if setdot {
		ed.end = len(lines)
		ed.start = ed.end
		ed.dot = ed.end
	}
	if !ed.silent && printsiz {
		fmt.Fprintln(ed.err, siz)
	}
	return lines, nil
}

// writeFile function will attempt to write the lines from index 'start'
// to 'end' in the file specified by 'path.' If successful, the current
// buffer will no longer be considered dirty.
func (ed *Editor) writeFile(start, end int, path string) error {
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
	ed.dirty = false
	if !ed.silent {
		fmt.Fprintln(ed.err, siz)
	}
	return err
}

// appendFile will open the file 'path' and append the lines starting
// at index 'start' until 'end.' If successful, the current buffer
// will no longer be considered dirty.
func (ed *Editor) appendFile(start, end int, path string) error {
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
	ed.dirty = false
	if !ed.silent {
		fmt.Fprintln(ed.err, siz)
	}
	return err
}

// shell runs the 'command' in /bin/sh and returns the standard output.
// It will replace any unescaped '%' with the name of the current buffer.
func (ed *Editor) shell(command string) ([]string, error) {
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
			if ed.path == "" {
				return output, ErrNoFileName
			}
			ctok = cs.Scan()
			parsed += ed.path
		}
		ctok = cs.Scan()
	}
	cmd := exec.Command(DefaultShell, "-c", parsed)
	stdout, err := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return output, err
	}
	defer stdout.Close()
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

// scanString will advance the tokenizer, scanning the input buffer
// until it reaches EOF, and return the collected tokens as a string.
// Newlines (\n) and carriage returns (\r) are ignored.
func (ed *Editor) scanString() string {
	var str string
	for ed.tok != scanner.EOF {
		if ed.tok != '\n' && ed.tok != '\r' {
			str += string(ed.tok)
		}
		ed.tok = ed.s.Scan()
	}
	return str
}

// scanStringUntil will advance the tokenizer, scanning the input
// buffer until it reaches the delimiter 'delim' or EOF, and return
// the collected tokens as a string.  Newlines (\n) and carriage returns
// (\r) are ignored.
func (ed *Editor) scanStringUntil(delim rune) string {
	var str string
	for ed.tok != scanner.EOF && ed.tok != delim {
		if ed.tok != '\n' && ed.tok != '\r' {
			str += string(ed.tok)
		}
		ed.tok = ed.s.Scan()
	}
	return str
}

// scanNumber attempts to lex a integer.
func (ed *Editor) scanNumber() (int, error) {
	var n, start, end int
	var err error
	start = ed.s.Position.Offset
	for unicode.IsDigit(ed.tok) {
		ed.tok = ed.s.Scan()
	}
	end = ed.s.Position.Offset
	num := string(ed.input[start:end])
	n, err = strconv.Atoi(num)
	return n, err
}

// skipWhitespace advances the internal tokenizer until the
// current token is not a white space, tab indent, or a newline.
func (ed *Editor) skipWhitespace() {
	for ed.tok == ' ' || ed.tok == '\t' || ed.tok == '\n' {
		ed.tok = ed.s.Scan()
	}
}

// undo undoes the last command and restores the current address
// to what it was before the last command.
func (ed *Editor) undo() (err error) {
	if len(ed.undohist) < 1 {
		return ErrNothingToUndo
	}
	var operation = ed.undohist[len(ed.undohist)-1]
	ed.undohist = ed.undohist[:len(ed.undohist)-1]
	var e int
	for n := len(operation) - 1; n >= 0; n-- {
		op := operation[n]
		switch op.typ {
		case undoDelete:
			ed.Lines = append(ed.Lines[:op.start], ed.Lines[op.end:]...)
		case undoAdd:
			ed.Lines = append(ed.Lines[:op.start-1], append(op.lines, ed.Lines[op.end:]...)...)
		}
		if op.d > 0 {
			op.end = op.d
		}
		if op.end > len(ed.Lines) {
			op.end = len(ed.Lines)
		}
		e = op.end
	}
	ed.start = e
	ed.end = e
	ed.dot = e
	return nil
}

// Do reads a command from `stdin` and executes the command.
// The output is printed to `stdout` and errors to `stderr`.
func (ed *Editor) Do() error {
	if err := ed.readInput(ed.in); err != nil {
		return err
	}
	if err := ed.parseRange(); err != nil {
		ed.error = err
		if !ed.printErrors {
			return ErrDefault
		}
		return err
	}
	if err := ed.doCommand(); err != nil {
		ed.error = err
		if !ed.printErrors {
			return ErrDefault
		}
		return err
	}
	return nil
}
