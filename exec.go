package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type cmd func(ed *Editor) error

var cmds map[rune]cmd

func init() {
	cmds = map[rune]cmd{
		'a':  cmdAppend,
		'c':  cmdChange,
		'd':  cmdDelete,
		'E':  cmdEdit,
		'e':  cmdEdit,
		'f':  cmdFilename,
		'V':  cmdGlobal,
		'G':  cmdGlobal,
		'v':  cmdGlobal,
		'g':  cmdGlobal,
		'H':  cmdHelp,
		'h':  cmdHelp,
		'i':  cmdInsert,
		'j':  cmdJoin,
		'k':  cmdMark,
		'l':  cmdPrint,
		'n':  cmdPrint,
		'p':  cmdPrint,
		'm':  cmdMove,
		'P':  cmdPrompt,
		'Q':  cmdQuit,
		'q':  cmdQuit,
		'r':  cmdRead,
		's':  cmdSubstitute,
		't':  cmdTransfer,
		'u':  cmdUndo,
		'W':  cmdWrite,
		'w':  cmdWrite,
		'z':  cmdScroll,
		'=':  cmdLineCount,
		'!':  cmdShell,
		'\n': cmdNone,
		EOF:  cmdNone,
	}
}

func (ed *Editor) exec() error {
	ed.skipWhitespace()
	if cmd, ok := cmds[ed.token()]; ok {
		return cmd(ed)
	}
	return ErrUnknownCmd
}

func cmdAppend(ed *Editor) error {
	ed.consume()
	if err := ed.getSuffix(); err != nil {
		return err
	}
	return ed.append(ed.second)
}

func cmdChange(ed *Editor) error {
	ed.consume()
	if err := ed.getSuffix(); err != nil {
		return err
	}
	if err := ed.validate(ed.dot, ed.dot); err != nil {
		return err
	}
	ed.delete(ed.first, ed.second)
	return ed.append(ed.dot)
}

func cmdDelete(ed *Editor) error {
	ed.consume()
	if err := ed.getSuffix(); err != nil {
		return err
	}
	if err := ed.validate(ed.dot, ed.dot); err != nil {
		return err
	}
	ed.delete(ed.first, ed.second)
	if ed.dot+1 < len(ed.file.lines) {
		ed.dot++
	}
	return nil
}

func cmdEdit(ed *Editor) error {
	r := ed.token()
	ed.consume()
	if ed.dirty && r == 'e' {
		ed.dirty = false
		return ErrFileModified
	}
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	} else if !unicode.IsSpace(ed.token()) {
		return ErrUnexpectedCmdSuffix
	}
	// if err := ed.getSuffix(); err != nil {
	// 	return err
	// }
	ed.skipWhitespace()
	ed.delete(1, len(ed.file.lines))
	return ed.read(ed.scanString())
}

func cmdFilename(ed *Editor) error {
	ed.consume()
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	}
	if !unicode.IsSpace(ed.token()) && ed.token() != EOF {
		return ErrUnexpectedCmdSuffix
	}
	ed.skipWhitespace()
	path := ed.scanString()
	if len(path) > 0 && path[0] == '!' {
		return ErrInvalidRedirection
	}
	var err error
	path, err = ed.validatePath(path)
	if err != nil {
		return err
	}
	ed.file.path = strings.Replace(path, "\\!", "!", -1)
	fmt.Fprintln(ed.stdout, ed.file.path)
	return nil
}

func cmdGlobal(ed *Editor) error {
	var err error
	r := ed.token()
	ed.consume()
	if ed.g {
		return ErrCannotNestGlobal
	} else if err := ed.validate(1, len(ed.file.lines)); err != nil {
		return err
	} else if err := ed.buildList(r); err != nil {
		return err
	}
	ed.g = true
	var (
		interactive = (r == 'G' || r == 'V')
		cmdlist     string
	)
	if !interactive {
		cmdlist, err = ed.cmdList()
		if err != nil {
			return err
		}
		if cmdlist == "" {
			cmdlist = "p"
		}
	}
	defer func() { ed.g = false }()
	gs := ed.cs
	nl := len(ed.file.lines)
	for _, i := range ed.list {
		ed.dot = i - (nl - len(ed.file.lines))
		if interactive {
			if err := ed.display(ed.dot, ed.dot, gs); err != nil {
				return err
			}
			cmdlist, err = ed.cmdList()
			if err != nil {
				return err
			}
			if cmdlist == "" {
				continue
			} else if cmdlist == "&" {
				if ed.gcmd == "" {
					return ErrNoPreviousCmd
				}
				cmdlist = ed.gcmd
			}
		}
		ed.doInput(cmdlist)
		if err := ed.parse(); err != nil {
			return err
		}
		if err := ed.exec(); err != nil {
			return err
		}
		if err := ed.display(ed.dot, ed.dot, ed.cs); err != nil {
			return err
		}
	}
	ed.cs = 0
	return nil
}

func cmdHelp(ed *Editor) error {
	r := ed.token()
	ed.consume()
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	if r == 'H' {
		ed.verbose = !ed.verbose
	}
	return ed.err
}

func cmdInsert(ed *Editor) error {
	ed.consume()
	if ed.second == 0 {
		ed.second = 1
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	return ed.append(ed.second - 1)
}

func cmdJoin(ed *Editor) error {
	ed.consume()
	if err := ed.validate(ed.dot, ed.dot+1); err != nil {
		return err
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	if ed.first != ed.second {
		// TODO: undo
		ed.file.join(ed.first, ed.second)
		ed.dot = ed.second
		ed.dirty = true
	}
	return nil
}

func cmdMark(ed *Editor) error {
	ed.consume()
	if ed.second == 0 {
		return ErrInvalidAddress
	}
	r := ed.token()
	ed.consume()
	if err := ed.getSuffix(); err != nil {
		return err
	}
	if !unicode.IsLower(r) || int(r-'a') >= len(ed.mark) {
		return ErrInvalidMark
	}
	ed.mark[r-'a'] = ed.second
	return nil
}

func cmdPrint(ed *Editor) error {
	if err := ed.validate(ed.dot, ed.dot); err != nil {
		return err
	}
	switch ed.token() {
	case 'l':
		ed.cs |= suffixList
	case 'n':
		ed.cs |= suffixEnumerate
	case 'p':
		ed.cs |= suffixPrint
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	return ed.display(ed.first, ed.second, ed.cs)
}

func cmdMove(ed *Editor) error {
	ed.consume()
	if err := ed.validate(ed.dot, ed.dot); err != nil {
		return err
	}
	addr, err := ed.getThirdAddr()
	if err != nil {
		return err
	}
	if ed.first <= addr && addr < ed.second {
		return ErrInvalidDestination
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	// TODO: undo move
	ed.dot = ed.file.move(ed.first, ed.second, addr)
	return nil
}

func cmdPrompt(ed *Editor) error {
	ed.consume()
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	ed.prompt = !ed.prompt
	return nil
}

func cmdQuit(ed *Editor) error {
	r := ed.token()
	ed.consume()
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	if r == 'q' && ed.dirty {
		ed.dirty = false
		return ErrFileModified
	}
	os.Exit(0)
	return nil
}

func cmdRead(ed *Editor) error {
	ed.consume()
	if !unicode.IsSpace(ed.token()) && ed.token() != EOF {
		return ErrUnexpectedCmdSuffix
	} else if ed.addrc == 0 {
		ed.second = len(ed.file.lines)
	}
	ed.skipWhitespace()
	path := ed.scanString()
	var err error
	path, err = ed.validatePath(path)
	if err != nil {
		return err
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	return ed.read(path)
}

func cmdSubstitute(ed *Editor) error {
	var err error
	ed.consume()
	nth, sflags := 1, 0
	const (
		subRepeatLast = 2 << iota
		subComplementGlobal
		subComplementPrint
		subLastRegex
	)
	for {
		r := ed.token()
		switch {
		case r == '\n', r == EOF: // repeat last substitution
			sflags |= subRepeatLast
			ed.consume()
		case r == 'g': // replace all occurences
			sflags |= subComplementGlobal
			ed.consume()
		case r == 'p': // print
			sflags |= subComplementPrint
			ed.consume()
		case r == 'r': // use last regex
			sflags |= subLastRegex
			ed.consume()
		case unicode.IsDigit(r): // replace nth
			nth, err = strconv.Atoi(string(r))
			if err != nil {
				return err
			}
			sflags |= subRepeatLast
			ed.consume()
			// override global substitution
		default:
			if sflags > 0 {
				return ErrInvalidCmdSuffix
			}
		}
		if sflags < 1 || r == '\n' || r == EOF {
			break
		}
	}
	if sflags > 0 && ed.re == nil {
		return ErrNoPrevPattern
	}
	delim := ed.token()
	ed.consume()

	re := ed.re
	if sflags&subLastRegex == 0 {
		search, eof := ed.scanStringUntil(delim)
		if !eof && ed.token() == delim {
			ed.consume()
		}
		if search == "" {
			if ed.re == nil {
				return ErrNoPrevPattern
			}
		} else {
			re, err = regexp.Compile(search)
			if err != nil {
				return err
			}
		}
	}
	replace := ed.replace
	var eof bool
	if sflags == 0 {
		replace, eof = ed.scanStringUntil(delim)
		if replace == "%" {
			if ed.replace == "" {
				return ErrNoPreviousSub
			}
			replace = ed.replace
		}
		if !eof {
			ed.consume()
		} else {
			sflags |= subComplementPrint
		}
	}
	if sflags&subComplementGlobal > 0 {
		nth = -1
	}
	if sflags&subComplementPrint > 0 {
		ed.cs |= suffixPrint
		ed.cs &= ^(suffixList | suffixEnumerate)
	}
	for !eof {
		r := ed.token()
		switch {
		case r == 'g':
			nth = -1
			ed.consume()
			continue
		case unicode.IsDigit(r):
			nth, err = strconv.Atoi(string(r))
			if err != nil {
				return err
			}
			sflags |= subRepeatLast
			ed.consume()
			continue
		default:
			if err := ed.getSuffix(); err != nil {
				return err
			}
		}
		break
	}
	if err := ed.validate(ed.dot, ed.dot); err != nil {
		return err
	}
	return ed.substitute(re, replace, nth)
}

func cmdTransfer(ed *Editor) error {
	ed.consume()
	if err := ed.validate(ed.dot, ed.dot); err != nil {
		return err
	}
	addr, err := ed.getThirdAddr()
	if err != nil {
		return err
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	lc := ed.file.yank(ed.first, ed.second, addr)
	ed.second = lc
	ed.dot = addr + lc
	ed.dirty = true
	return nil
}

func cmdUndo(ed *Editor) error {
	ed.consume()
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	return ed.undo.pop()
}

func cmdWrite(ed *Editor) error {
	r := ed.token()
	ed.consume()
	quit := ed.token()
	path := ""
	if !unicode.IsSpace(quit) && quit != 'Q' && quit != 'q' && quit != EOF {
		return ErrUnexpectedCmdSuffix
	}
	ed.skipWhitespace()
	if ed.token() != EOF && ed.token() != '\n' {
		path = ed.scanString()
	}
	var err error
	path, err = ed.validatePath(path)
	if err != nil {
		return err
	}
	if ed.addrc == 0 && len(ed.file.lines) < 1 {
		ed.first, ed.second = 0, 0
	} else if err := ed.validate(1, len(ed.file.lines)); err != nil {
		return err
	}
	if err := ed.getSuffix(); err != nil {
		return err
	}
	return ed.write(path, r, quit, ed.first, ed.second)
}

func cmdScroll(ed *Editor) error {
	ed.consume()
	ed.first = 1
	if err := ed.validate(ed.first, ed.dot+1); err != nil {
		return err
	} else if unicode.IsDigit(ed.token()) {
		n, err := ed.scanNumber()
		if err != nil {
			return err
		}
		ed.scroll = n
	}
	ed.cs = suffixPrint
	if err := ed.getSuffix(); err != nil {
		return err
	}
	scroll := len(ed.file.lines)
	if ed.second+ed.scroll < len(ed.file.lines) {
		scroll = ed.second + ed.scroll
	}
	return ed.display(ed.second, scroll, ed.cs)
}

func cmdLineCount(ed *Editor) error {
	ed.consume()
	if err := ed.getSuffix(); err != nil {
		return err
	}
	n := ed.second
	if ed.addrc < 1 {
		n = len(ed.file.lines)
	}
	fmt.Fprintln(ed.stdout, n)
	return nil
}

func cmdShell(ed *Editor) error {
	ed.consume()
	if ed.addrc > 0 {
		return ErrUnexpectedAddress
	}
	if ed.token() == EOF || ed.token() == '\n' {
		return ErrNoCmd
	}
	ed.skipWhitespace()
	cmd := ed.scanString()
	output, err := ed.shell(cmd)
	if err != nil {
		return err
	}
	output = append(output, "!")
	fmt.Fprintln(ed.stdout, strings.Join(output, "\n"))
	return nil
}

func cmdNone(ed *Editor) error {
	ed.first = 1
	if err := ed.validate(ed.first, ed.dot+1); err != nil {
		return err
	}
	return ed.display(ed.second, ed.second, suffixPrint)
}
