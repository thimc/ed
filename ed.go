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
	"strings"
	"syscall"
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
	ErrInterrupt           = errors.New("interrupt")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidCmdSuffix    = errors.New("invalid command suffix")
	ErrInvalidDestination  = errors.New("invalid destination")
	ErrInvalidMark         = errors.New("invalid mark character")
	ErrInvalidNumber       = errors.New("number out of range")
	ErrInvalidPatternDelim = errors.New("invalid pattern delimiter")
	ErrInvalidRedirection  = errors.New("invalid redirection")
	ErrNoCmd               = errors.New("no command")
	ErrNoFileName          = errors.New("no current filename")
	ErrNoMatch             = errors.New("no match")
	ErrNoPrevPattern       = errors.New("no previous pattern")
	ErrNoPreviousCmd       = errors.New("no previous command")
	ErrNoPreviousSub       = errors.New("no previous substitution")
	ErrNothingToUndo       = errors.New("nothing to undo")
	ErrUnexpectedAddress   = errors.New("unexpected address")
	ErrUnexpectedCmdSuffix = errors.New("unexpected command suffix")
	ErrUnknownCmd          = errors.New("unknown command")
	ErrZero                = errors.New("0")
)

type Editor struct {
	path  string   // file path
	dirty bool     // modified
	Lines []string // File buffer
	mark  [25]int  // a to z
	dot   int      // current position

	start int // start position
	end   int // end position
	addrc int // number of addresses in the current input

	*Tokenizer // user input

	undohist    [][]undoAction // undo history
	globalUndo  []undoAction   // undo actions caught during global cmds
	g           bool           // global command state
	error       error          // previous error
	scroll      int            // previous scroll value
	search      string         // previous search criteria for /, ? or s
	replacestr  string         // previous s replacement
	showPrompt  bool           // toggle for displaying the prompt
	prompt      string         // user prompt
	shellCmd    string         // previous command for !
	globalCmd   string         // previous command used by g, G, v and V
	printErrors bool           // toggle errors
	silent      bool           // chatty

	sighupch chan os.Signal
	sigintch chan os.Signal

	in  io.Reader // standard input
	out io.Writer // standard output
	err io.Writer // standard error
}

// New creates a new instance of the Ed editor. It defaults to reading
// user input from the operating systems standard input, printing to
// standard output and errors to standard error. Signal handlers are
// created.
func New(opts ...OptionFunc) *Editor {
	ed := &Editor{
		sigintch: make(chan os.Signal, 1),
		sighupch: make(chan os.Signal, 1),
		in:       os.Stdin,
		out:      os.Stdout,
		err:      os.Stderr,
	}
	for _, opt := range opts {
		opt(ed)
	}
	signal.Notify(ed.sigintch, syscall.SIGINT, os.Interrupt)
	signal.Notify(ed.sighupch, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for range ed.sighupch {
			if ed.dirty {
				ed.writeFile(false, 1, len(ed.Lines), DefaultHangupFile)
			}
			ed.error = ErrInterrupt
		}
	}()
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
// func (ed *Editor) readInput(r io.Reader) error {
// br := bufio.NewReader(r)
// ln, err := br.ReadString('\n')
// if err != nil {
// 	if errors.Is(err, io.EOF) {
// 		return ErrInterrupt
// 	}
// 	return err
// }
// if len(ln) > 1 {
// 	ln = ln[:len(ln)-1]
// }
// 	return nil
// }

// readFile checks if the 'path' starts with a '!' and if so executes
// what it presumes to be a valid shell command in sh(1). If readFile
// deems 'path' not to be a shell expression it will attempt to open
// 'path' like a regular file.  If no error occurs and 'setdot' is
// true, the cursor positions are set to the last line of the buffer.
// If 'printsiz' is set to true, the size in bytes is printed to the
// 'err io.Writer'.
func (ed *Editor) readFile(path string, setdot bool, printsiz bool) ([]string, error) {
	var (
		siz   int64
		cmd   bool
		lines []string
	)
	if len(path) > 0 {
		cmd = (path[0] == '!')
		if cmd {
			path = path[1:]
		}
	}
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
			if !ed.printErrors {
				err = ErrZero
			}
			return lines, err
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

// writeFile writes (or appends) the buffer lines from `start` to
// `end` to `path`. It clears the `dirty` flag if successful. `append`
// determines if the data will be appended to the existing or if the
// file will be overriden.
func (ed *Editor) writeFile(append bool, start, end int, path string) error {
	var (
		file *os.File
		err  error
	)
	if append {
		file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	} else {
		file, err = os.Create(path)
	}
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

// shell runs the 'command' in a subshell (`defaultShell`) and returns
// the output.  It will replace any unescaped '%' with the name of the
// current buffer.
func (ed *Editor) shell(cmd string) ([]string, error) {
	t := NewTokenizer(strings.NewReader(cmd))
	t.token()
	var parsed string
	for t.tok != EOF {
		parsed += string(t.tok)
		if t.tok != '\\' && t.peek() == '%' {
			if ed.path == "" {
				return nil, ErrNoFileName
			}
			t.token()
			parsed += ed.path
		}
		t.token()
	}
	c := exec.Command(DefaultShell, "-c", parsed)
	output, err := c.Output()
	if err != nil {
		return nil, err
	}
	var (
		lines  = bytes.Split(output, []byte("\n"))
		result = make([]string, len(lines))
	)
	for i, line := range lines {
		result[i] = string(line)
	}
	return result[:len(result)-1], nil
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

// Do reads a command from `in` and executes the command.
// The output is printed to `out` and errors to `err`.
func (ed *Editor) Do() error {
	ed.Tokenizer = NewTokenizer(ed.in)
	ed.token()
	if err := ed.parse(); err != nil && ed.tok == EOF {
		ed.error = err
		if !ed.printErrors {
			return ErrDefault
		}
		return err
	}

	log.Printf("\033[1mstart=%d, end=%d, dot=%d, addrc=%d [tok:%q, eof:%t, len:%d]\n\033[0m",
		ed.start, ed.end, ed.dot, ed.addrc, ed.tok, ed.tok == EOF, len(ed.Lines))

	if err := ed.do(); err != nil {
		ed.error = err
		if !ed.printErrors {
			return ErrDefault
		}
		return err
	}
	return nil
}
