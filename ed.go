package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
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
	ErrUnexpectedEOF       = errors.New("unexpected end-of-file")
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
	position   // cursors
	*tokenizer // user input

	path     string // full path to file
	modified bool
	silent   bool // suppress diagnostics
	scripted bool // terminal check
	lines    []string
	mark     [25]int // a to z

	lineno int // line number in script

	cs cmdSuffix // command suffix
	ss subSuffix // substitution suffix

	undohist [][]undoAction // undo history

	globalUndo []undoAction // undo actions caught during global mode
	g          bool         // global command state
	list       []int        // indices of marked lines

	error      error          // previous error
	globalCmd  string         // previous command used by g, G, v and V
	re         *regexp.Regexp // previous regex for /, ? and s
	replacestr string         // previous s replacement
	scroll     int            // previous scroll value
	shellCmd   string         // previous command for !

	showPrompt  bool   // toggle for displaying the prompt
	prompt      string // user prompt
	printErrors bool   // toggle errors

	sighupch chan os.Signal
	sigintch chan os.Signal

	in  io.Reader // input
	out io.Writer // output
	err io.Writer // error
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
		lineno:   1,
	}
	for _, opt := range opts {
		opt(ed)
	}
	signal.Notify(ed.sigintch, syscall.SIGINT, os.Interrupt)
	signal.Notify(ed.sighupch, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for range ed.sighupch {
			if ed.modified && len(strings.Join(ed.lines, "")) > 0 {
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

// WithSilent overrides the default silent mode of the editor. This mode
// is meant to be used with ed scripts.
func WithSilent(b bool) OptionFunc {
	return func(ed *Editor) {
		ed.silent = b
	}
}

// WithScripted determines if ed should treat the [ed.in] reader as if
// the data is being piped from a file. If [t] is set to true, ed will
// try to parse as many lines of commands as possible. If any errors
// are encountered they are wrapped into errors that help the user
// debug script errors.
func WithScripted(t bool) OptionFunc {
	return func(ed *Editor) {
		ed.scripted = t
	}
}

// wrapError takes an error and stores it, it returns the default
// error, [ErrDefault], if [ed.printErrors] is set to false. It returns
// a [explainError] if the 'h' command was just executed.  wrapError
// always returns the actual error along with additional information
// if the editor is running in scripted mode.
func (ed *Editor) wrapError(err error) error {
	ed.error = err
	if ex, ok := err.(explainError); ok {
		return ex
	}
	if !ed.printErrors {
		return ErrDefault
	}
	if ed.scripted {
		return fmt.Errorf("script, line %d: %s", ed.lineno, ed.error)
	}
	return ed.error
}

// Do reads expects data from the `in io.Reader` and reads until newline
// or EOF.  After which it parses the user input for addresses (if any)
// and finally executes the command (if any). If the prompt is enabled
// it is printed to [ed.out] before prompting the user.
func (ed *Editor) Do() error {
	var err error
	if ed.showPrompt {
		fmt.Fprint(ed.out, ed.prompt)
	}
	if !ed.scripted || ed.tokenizer == nil {
		ed.tokenizer = newTokenizer(ed.in)
	}
	if ed.consume() == EOF {
		if !ed.modified {
			return io.EOF
		}
		ed.tok = 'q'
	}
	if err = ed.parse(); err != nil {
		return ed.wrapError(err)
	}
	if err = ed.do(); err != nil {
		return ed.wrapError(err)
	}
	if ed.cs > 0 {
		if err = ed.displayLines(ed.dot, ed.dot, ed.cs); err != nil {
			return ed.wrapError(err)
		}
	}
	ed.error = err
	if ed.scripted {
		ed.lineno++
		return ed.Do()
	}
	return nil
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
	t.consume()
	var parsed string
	for t.tok != EOF {
		parsed += string(t.tok)
		if t.tok != '\\' && t.peek() == '%' {
			if ed.path == "" {
				return nil, ErrNoFileName
			}
			t.consume()
			parsed += ed.path
		}
		t.consume()
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
	ed.shellCmd = cmd
	return result[:len(result)-1], nil
}

// interrupt is the handler for when a SIGINT has been received.
// It sets the internal error and prints [ErrDefault] to `err`.
func (ed *Editor) interrupt() error {
	ed.error = ErrInterrupt
	fmt.Fprintln(ed.err, ErrDefault)
	return ErrInterrupt
}
