package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

const (
	// The default value of the user prompt which can be overriden with
	// the `P` command.
	DefaultPrompt = "*"
	// The default file name for when the editor receives a SIGHUP which
	// causes ed to dump all the contents of the buffer to the hangup file.
	DefaultHangupFile = "ed.hup"
	// The default file path to the shell used by `!` commands.
	DefaultShell = "/bin/sh"
)

type undoType int

const (
	undoAdd    undoType = iota // undoAdd adds lines
	undoDelete                 // undoDelete deletes lines
)

type undoAction struct {
	typ   undoType
	start int
	end   int
	dot   int
	lines []string
}

// Ed is limited to displaying these error messages with the exception
// of regular expression errors.
var (
	ErrDefault             = errors.New("?") // descriptive error message, don't you think?
	ErrCryptUnavailable    = errors.New("crypt unavailable")
	ErrCannotNestGlobal    = errors.New("cannot nest global commands")
	ErrCannotOpenFile      = errors.New("cannot open input file")
	ErrCannotWriteFile     = errors.New("cannot write file")
	ErrCannotCloseFile     = errors.New("cannot close input file")
	ErrCannotReadFile      = errors.New("cannot read input file")
	ErrDestinationExpected = errors.New("destination expected")
	ErrFileModified        = errors.New("warning: file modified")
	ErrInterrupt           = errors.New("interrupt")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidCmdSuffix    = errors.New("invalid command suffix")
	ErrInvalidDestination  = errors.New("invalid destination")
	ErrInvalidMark         = errors.New("invalid mark character")
	ErrInvalidFileName     = errors.New("invalid filename")
	ErrInvalidNumber       = errors.New("number out of range")
	ErrInvalidPatternDelim = errors.New("invalid pattern delimiter")
	ErrInvalidRedirection  = errors.New("invalid redirection")
	ErrNoCmd               = errors.New("no command")
	ErrNumberOutOfRange    = errors.New("number out of range")
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

// position contains the cursor positions used by ed.
type position struct {
	start, end, dot, addrc int
}

// Editor contains the internal data structures needed for ed.
type Editor struct {
	position
	*tokenizer // user input

	path     string // full path to file
	modified bool
	scripted bool
	lines    []string
	mark     [25]int // a to z

	cs cmdSuffix // command suffix
	ss subSuffix // substitution suffix

	undohist    [][]undoAction // undo history
	globalUndo  []undoAction   // undo actions caught during global mode
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

	sighupch chan os.Signal
	sigintch chan os.Signal

	in  io.Reader // standard input
	out io.Writer // standard output
	err io.Writer // standard error
}

// New creates a new Editor. If no options are passed the editor
// defaults to reading from `os.Stdin` and printing to `os.Stdout` and
// `os.Stderr`.
// Signal handlers for SIGINT and SIGHUP are automatically created.
func New(opts ...OptionFunc) *Editor {
	ed := &Editor{
		sigintch: make(chan os.Signal, 1),
		sighupch: make(chan os.Signal, 1),
		in:       os.Stdin,
		out:      os.Stdout,
		err:      os.Stderr,
		prompt:   DefaultPrompt,
	}
	for _, opt := range opts {
		opt(ed)
	}
	signal.Notify(ed.sigintch, syscall.SIGINT, os.Interrupt)
	signal.Notify(ed.sighupch, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for range ed.sighupch {
			if ed.modified {
				ed.writeFile(DefaultHangupFile, 'w', 1, len(ed.lines))
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
		if ed.prompt == "" {
			ed.prompt = DefaultPrompt
		}
		ed.showPrompt = (s != "")
	}
}

// WithFile sets the inital buffer of the editor. On error it ignores
// the state of the `H` command and prints the actual error to the
// editors stderr `io.Writer`.
func WithFile(path string) OptionFunc {
	return func(ed *Editor) {
		if err := ed.readFile(path); err != nil {
			fmt.Fprintln(ed.err, err)
		}
	}
}

// WithScripted overrides the default silent mode of the editor. This mode
// is meant to be used with ed scripts.
func WithScripted(b bool) OptionFunc {
	return func(ed *Editor) {
		ed.scripted = b
	}
}

// Do reads expects data from the `in io.Reader` and reads until newline
// or EOF.  After which it parses the user input for addresses (if any)
// and finally executes the command (if any). If the prompt is enabled
// it is printed to the `err io.Writer` before any data is read. If the
// internal state of the editor is to not explain errors then if any
// errors are encountered they are automatically defaulted to [ErrDefault].
// Do returns the error io.EOF if the in reader is empty.
func (ed *Editor) Do() error {
	if ed.showPrompt {
		fmt.Fprint(ed.out, ed.prompt)
	}
	if ed.tokenizer == nil {
		ed.tokenizer = newTokenizer(ed.in)
	}
	ed.token()
	if ed.tok == EOF {
		if ed.scripted || !ed.modified {
			return io.EOF
		}
		ed.tok = 'q'
	}
	if err := ed.parse(); err != nil {
		ed.error = err
		if !ed.printErrors {
			return ErrDefault
		}
		return ed.error
	}
	if ed.error = ed.do(); ed.error != nil {
		if !ed.printErrors {
			return ErrDefault
		}
		return ed.error
	}
	if ed.cs > 0 {
		if ed.error = ed.displayLines(ed.dot, ed.dot, ed.cs); ed.error != nil {
			if !ed.printErrors {
				return ErrDefault
			}
		}
	}
	if ed.scripted && ed.tok == '\n' && ed.peek() != EOF {
		return ed.Do()
	}
	return ed.error
}

// shell runs the [command] in a subshell ([DefaultShell]) and returns
// the output.  It will replace any unescaped '%' with the name of the
// current buffer.
func (ed *Editor) shell(cmd string) ([]string, error) {
	if cmd[0] == '!' {
		cmd = cmd[1:]
	}
	if cmd == "" {
		if ed.shellCmd == "" {
			return nil, ErrNoPreviousCmd
		}
		cmd = ed.shellCmd
	}

	t := newTokenizer(strings.NewReader(cmd))
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

// interrupt is the handler for when a SIGINT has been received.
// It sets the internal error and prints [ErrDefault] to `err`.
func (ed *Editor) interrupt() error {
	ed.error = ErrInterrupt
	fmt.Fprintln(ed.err, ErrDefault)
	return ErrInterrupt
}
