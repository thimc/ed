package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Ed is limited to displaying these error messages with the exception
// of regular expression errors.
var (
	ErrDefault             = errors.New("?") // descriptive error message, don't you think?
	ErrCannotCloseFile     = errors.New("cannot close input file")
	ErrCannotNestGlobal    = errors.New("cannot nest global commands")
	ErrCannotOpenFile      = errors.New("cannot open input file")
	ErrCannotReadFile      = errors.New("cannot read input file")
	ErrCannotWriteFile     = errors.New("cannot write file")
	ErrCryptUnavailable    = errors.New("crypt unavailable")
	ErrDestinationExpected = errors.New("destination expected")
	ErrFileModified        = errors.New("warning: file modified")
	ErrInterrupt           = errors.New("interrupt")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidCmdSuffix    = errors.New("invalid command suffix")
	ErrInvalidDestination  = errors.New("invalid destination")
	ErrInvalidFileName     = errors.New("invalid filename")
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
	ErrNumberOutOfRange    = errors.New("number out of range")
	ErrUnexpectedAddress   = errors.New("unexpected address")
	ErrUnexpectedCmdSuffix = errors.New("unexpected command suffix")
	ErrUnexpectedEOF       = errors.New("unexpected end-of-file")
	ErrUnknownCmd          = errors.New("unknown command")
	ErrZero                = errors.New("0")
)

type suffix int

const (
	suffixPrint suffix = 1 << iota
	suffixList
	suffixEnumerate
)

const DefaultShell = "/bin/sh"
const DefaultHangupFile = "ed.hup"
const DefaultPrompt = "*"

type Editor struct {
	file
	cursor
	undo
	input

	re      *regexp.Regexp // previous regex
	replace string         // previous replacement text
	scroll  int            // previous scroll value
	err     error          // previous error
	gcmd    string         // previous global command

	g    bool  // global command state
	list []int // indices marked by the global command

	prompt  bool           // state for rendering the prompt
	up      string         // user prompt
	verbose bool           // toggle verbose errors
	silent  bool           // suppress diagnostics
	script  bool           // stdin is a file
	lc      int            // line count (script mode)
	binary  bool           // TODO(thimc): implement "binary mode" which replaces every NULL character with a newline. When this mode is enabled ed should not append a newline on reading/writing.
	sigch   chan os.Signal // signal handlers

	cs suffix // command suffix

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type Option func(*Editor)

func WithStdin(stdin io.Reader) Option {
	return func(ed *Editor) {
		ed.stdin = stdin
		ed.input = input{Scanner: bufio.NewScanner(ed.stdin)}
	}
}

func WithStdout(stdout io.Writer) Option {
	return func(ed *Editor) { ed.stdout = stdout }
}

func WithStderr(stderr io.Writer) Option {
	return func(ed *Editor) { ed.stderr = stderr }
}

func WithSilent(t bool) Option {
	return func(ed *Editor) { ed.silent = t }
}

func WithPrompt(prompt string) Option {
	return func(ed *Editor) {
		ed.up = prompt
		ed.prompt = ed.up != ""
	}
}

func WithFile(path string) Option {
	return func(ed *Editor) {
		if err := ed.read(path); err != nil {
			ed.errorln(true, err)
		}
	}
}

func NewEditor(opts ...Option) *Editor {
	ed := &Editor{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		sigch:  make(chan os.Signal, 1),
		lc:     1,
	}
	if fi, err := os.Stdin.Stat(); err == nil {
		ed.script = fi.Mode()&os.ModeCharDevice == 0
	}
	for _, opt := range opts {
		opt(ed)
	}
	go ed.handleSignals()
	return ed
}

func (ed *Editor) validatePath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	if ed.path == "" {
		return "", ErrNoFileName
	}
	return ed.path, nil
}

func (ed *Editor) validate(f, s int) error {
	if ed.addrc == 0 {
		ed.first = f
		ed.second = s
	}
	if ed.first > ed.second || ed.first < 1 || ed.second > len(ed.file.lines) {
		return ErrInvalidAddress
	}
	return nil
}

func (ed *Editor) doPrompt() {
	if ed.prompt && ed.up != "" {
		fmt.Fprint(ed.stdout, ed.up)
	}
}

func (ed *Editor) errorln(verbose bool, err error) {
	if ed.token() == 'H' {
		return
	} else if ed.token() == 'h' {
		ed.consume()
		if ed.err != nil {
			fmt.Fprintln(ed.stderr, ed.err)
		}
		return
	}
	ed.err = err
	if verbose {
		if ed.script {
			fmt.Fprintf(ed.stderr, "script, line: %d: %s\n", ed.lc, ed.err)
			os.Exit(2)
		}
		fmt.Fprintln(ed.stderr, err)
		return
	}
	fmt.Fprintln(ed.stderr, ErrDefault)
}

func (ed *Editor) run() error {
	ed.doPrompt()
	if !ed.input.Scan() {
		if !ed.file.dirty {
			ed.input.pos = -1
			return nil
		}
		WithStdin(ed.stdin)(ed)
		ed.doInput("q")
	}
	if ed.script {
		ed.lc++
	}
	if err := ed.parse(); err != nil {
		return err
	}
	if err := ed.exec(); err != nil {
		return err
	}
	return ed.display(ed.dot, ed.dot, ed.cs)
}

func (ed *Editor) Run() {
	for {
		err := ed.run()
		if ed.input.pos < 0 {
			break
		}
		if err != nil {
			ed.errorln(ed.verbose, err)
			continue
		}
		ed.err = nil
	}
}

func (ed *Editor) getThirdAddr() (int, error) {
	start, end := ed.first, ed.second
	if err := ed.parse(); err != nil {
		return -1, err
	}
	if ed.addrc == 0 {
		return -1, ErrDestinationExpected
	}
	if ed.second < 0 || ed.second > len(ed.file.lines) {
		return -1, ErrInvalidAddress
	}
	addr := ed.second
	ed.first, ed.second = start, end
	return addr, nil
}

func (ed *Editor) read(path string) error {
	var (
		lines []string
		err   error
	)
	path, err = ed.validatePath(path)
	if err != nil {
		return err
	}
	if r, _ := utf8.DecodeRuneInString(path); r == '!' {
		if path[1:] == "" {
			return ErrNoCmd
		}
		lines, err = ed.shell(path[1:])
		if err != nil {
			return err
		}
		ed.file.append(ed.second, lines)
	} else {
		buf, err := os.ReadFile(path)
		if err != nil {
			return ErrCannotReadFile
		}
		lines = strings.Split(strings.TrimSuffix(string(buf), "\n"), "\n")
		n := ed.second
		if n >= len(ed.file.lines) {
			n = 0
		}
		ed.file = file{
			lines: append(ed.file.lines[:n], append(lines, ed.file.lines[n:]...)...),
			path:  path,
		}
		ed.undo.reset()
	}
	size := len(lines)
	for _, ln := range lines {
		size += len(ln)
	}
	ed.dot = ed.second + len(lines)
	if ed.dot >= len(ed.file.lines) {
		ed.dot = len(lines)
	}
	if !ed.silent {
		fmt.Fprintln(ed.stdout, size)
	}
	return nil
}

func (ed *Editor) append(dot int) error {
	for ed.Scan() {
		ln := ed.scanString()
		if ln == "." {
			break
		}
		ed.file.append(dot, []string{ln})
		dot++
		ed.undo.append(undoTypeDelete, cursor{first: dot, second: dot, dot: ed.dot}, ed.file.lines[dot-1:dot])
		ed.dot = dot
		ed.dirty = true
		if ed.script {
			ed.lc++
		}
	}
	ed.undo.store(ed.g)
	return nil
}

func (ed *Editor) delete(start, end int) {
	lines := make([]string, end-start+1)
	copy(lines, ed.file.lines[start-1:end])
	ed.undo.append(undoTypeAdd, cursor{first: start, second: end, dot: ed.dot}, lines)
	ed.file.delete(start, end)
	ed.dot = start - 1
	ed.dirty = true
}

func (ed *Editor) display(start, end int, flags suffix) error {
	if flags == 0 {
		return nil
	}
	if start < 1 {
		return ErrInvalidAddress
	}
	for start--; start != end; start++ {
		ed.dot = start + 1
		var ln string
		if flags&suffixEnumerate > 0 {
			ln = fmt.Sprintf("%d\t", ed.dot)
		}
		if flags&suffixList > 0 {
			quoted := strings.Replace(strconv.QuoteToASCII(ed.file.lines[start]), "$", "\\$", -1)
			ln += fmt.Sprintf("%s$", quoted[1:len(quoted)-1])
		} else {
			ln += ed.file.lines[start]
		}
		fmt.Fprintln(ed.stdout, ln)
	}
	ed.cs = 0
	return nil
}

func (ed *Editor) shell(args string) ([]string, error) {
	var sb strings.Builder
	count := utf8.RuneCountInString(args)
	for i := 0; i < count; {
		r, w := utf8.DecodeRuneInString(args[i:])
		if i+w < count {
			next, nw := utf8.DecodeRuneInString(args[i+w:])
			if next == '%' {
				i += w + nw
				if r == '\\' {
					sb.WriteRune(next)
				} else {
					var err error
					ed.path, err = ed.validatePath(ed.path)
					if err != nil {
						return nil, err
					}
					sb.WriteRune(r)
					sb.WriteString(ed.path)
				}
				continue
			}
		}
		i += w
		sb.WriteRune(r)
	}
	cmd := exec.Command(DefaultShell, "-c", sb.String())
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(string(output), "\n"), "\n"), err
}

func (ed *Editor) getSuffix() error {
	var ok bool
	r := ed.token()
	for !ok {
		r = ed.token()
		switch r {
		case 'n':
			ed.cs |= suffixEnumerate
			ed.consume()
		case 'l':
			ed.cs |= suffixList
			ed.consume()
		case 'p':
			ed.cs |= suffixPrint
			ed.consume()
		default:
			ok = true
		}
	}
	if !ed.input.eof() && r != '\n' {
		return ErrInvalidCmdSuffix
	}
	return nil
}

func (ed *Editor) buildList(g, interactive bool) error {
	delim := ed.token()
	if delim == ' ' || delim == '\n' || ed.input.eof() {
		return ErrInvalidPatternDelim
	}
	ed.consume()
	search, _ := ed.scanStringUntil(delim)
	if ed.token() == delim {
		ed.consume()
	}
	var (
		re  *regexp.Regexp
		err error
	)
	if search == "" {
		if ed.re == nil {
			return ErrNoPrevPattern
		}
		re = ed.re
	} else {
		re, err = regexp.Compile(search)
		if err != nil {
			return err
		}
	}
	if interactive {
		if err := ed.getSuffix(); err != nil {
			return err
		}
	}
	ed.list = []int{}
	for i := ed.first - 1; i < ed.second; i++ {
		if re.MatchString(ed.file.lines[i]) == g {
			ed.list = append(ed.list, i+1)
		}
	}
	ed.re = re
	return nil
}

func (ed *Editor) cmdList() (string, error) {
	var (
		sb   strings.Builder
		done bool
		ln   string
	)
	if !ed.input.eof() {
		ln = ed.scanString()
	}
	for {
		done = true
		if strings.HasSuffix(ln, "\\") {
			ln = strings.TrimSuffix(ln, "\\")
			done = false
			ed.consume()
		}
		sb.WriteString(ln)
		if !done {
			if !ed.input.Scan() {
				return "", ErrUnexpectedEOF
			}
			ln = ed.input.buf
			continue
		}
		break
	}
	return sb.String(), nil
}

func (ed *Editor) substitute(re *regexp.Regexp, replace string, nth int) error {
	var subs int
	for i := ed.first - 1; i < ed.second; i++ {
		if !re.MatchString(ed.file.lines[i]) {
			continue
		}
		submatch := re.FindAllStringSubmatch(ed.file.lines[i], -1)
		matches := re.FindAllStringIndex(ed.file.lines[i], -1)
		for mi, match := range matches {
			if nth > 0 && mi != nth-1 {
				continue
			}
			start, end := match[0], match[1]
			var sb strings.Builder
			sb.WriteString(ed.file.lines[i][:start])
			count := utf8.RuneCountInString(replace)
			for j := 0; j < count; {
				r, w := utf8.DecodeRuneInString(replace[j:])
				if j+w < count {
					next, nw := utf8.DecodeRuneInString(replace[j+w:])
					if r == '\\' && next == '&' {
						j += w + nw
						sb.WriteRune(next)
						continue
					} else if r == '\\' && unicode.IsDigit(next) {
						j += w + nw
						d, err := strconv.Atoi(string(next))
						if err != nil || d < 1 || d > len(submatch) {
							return ErrNumberOutOfRange
						}
						if d >= len(submatch[0]) {
							sb.WriteRune(next)
						} else if d > 0 {
							sb.WriteString(submatch[0][d])
						}
						continue
					}
				}
				if r == '&' {
					sb.WriteString(ed.file.lines[i][start:end])
					j += w
					continue
				}
				j += w
				sb.WriteRune(r)
			}
			// TODO(thimc): Handle embedded newlines in the replacement string.
			sb.WriteString(ed.file.lines[i][end:])
			ed.undo.append(undoTypeAdd, cursor{first: i + 1, second: i + 1, dot: ed.dot}, []string{ed.file.lines[i]})
			ed.undo.append(undoTypeDelete, cursor{first: i + 1, second: i + 1, dot: ed.dot}, []string{sb.String()})
			ed.file.lines[i] = sb.String()
			ed.dirty = true
			ed.dot = i + 1
			subs++
		}
	}
	ed.re = re
	ed.replace = replace
	if subs == 0 && !ed.g {
		return ErrNoMatch
	}
	ed.undo.store(ed.g)
	return ed.display(ed.dot, ed.dot, ed.cs)
}
