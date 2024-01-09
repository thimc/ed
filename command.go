package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"

	"os"
	"os/exec"
	"text/scanner"
)

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

func (ed *Editor) consumeWhitespace() {
	for ed.token() == ' ' || ed.token() == '\t' || ed.token() == '\n' {
		ed.nextToken()
	}
}

func (ed *Editor) ReadFile(path string) error {
	var siz int64
	file, err := os.Open(path)
	if err != nil {
		return ErrCannotOpenFile
	}
	stat, err := file.Stat()
	if err != nil {
		return ErrCannotOpenFile
	}
	siz = stat.Size()
	s := bufio.NewScanner(file)
	for s.Scan() {
		ed.Lines = append(ed.Lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return err
	}
	ed.Path = path
	ed.Dot = len(ed.Lines)
	ed.Start = ed.Dot
	ed.End = ed.Dot
	ed.addr = -1
	fmt.Fprintf(ed.err, "%d\n", siz)
	return nil
}

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

func (ed *Editor) DoCommand() error {
	log.Printf("Cmd='%c' (EOF=%t)\n", ed.token(), ed.token() == scanner.EOF)

	// FIXME: We might need to check the bounds in some of these commands
	// adding a ed.checkRanges() here will block the user from inserting
	// text if the start and end values are invalid.

	switch ed.token() {
	case 'a':
		for {
			line, err := ed.ReadInsert()
			if err != nil {
				ed.setupSignals()
				return nil
			}
			if line == "." {
				break
			}

			if len(ed.Lines) == ed.End {
				ed.Lines = append(ed.Lines, line)
				ed.End++
				continue
			}
			ed.Lines = append(ed.Lines[:ed.End+1], ed.Lines[ed.End:]...)
			ed.Lines[ed.End] = line
			ed.Dirty = true
		}
		return nil

	case 'c':
		ed.Dirty = true
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		ed.End = ed.Start - 1
		for {
			line, err := ed.ReadInsert()
			if err != nil {
				ed.setupSignals()
				return nil
			}
			if line == "." {
				break
			}
			ed.Lines = append(ed.Lines[:ed.End+1], ed.Lines[ed.End:]...)
			ed.Lines[ed.End] = line
			ed.End++
		}
		return nil

	case 'd':
		ed.Dirty = true
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		if ed.Start > len(ed.Lines) {
			ed.Start = len(ed.Lines)
		}
		ed.Dot = ed.Start
		ed.End = ed.Dot
		ed.Start = ed.Dot
		return nil

	case 'E':
		fallthrough
	case 'e':
		var uc bool = (ed.token() == 'E')
		ed.nextToken()
		ed.nextToken()
		var cmd bool
		if ed.token() == '!' {
			ed.nextToken()
			cmd = true
		}
		ed.consumeWhitespace()
		var fname string = ed.scanString()
		switch cmd {
		case true:
			if fname == "" && ed.Cmd != "" {
				fname = ed.Cmd
			}
			log.Printf("e command '%s'\n", fname)
			lines, err := ed.Shell(fname)
			if err != nil {
				return ErrZero
			}
			var siz int
			for i := range lines {
				siz += len(lines[i]) + 1
			}
			ed.Lines = lines
			ed.Dot = len(ed.Lines)
			ed.Start = ed.Dot
			ed.End = ed.Dot
			ed.addr = -1
			fmt.Fprintf(ed.err, "%d\n", siz)
		case false:
			if fname == "" && ed.Path == "" {
				return ErrNoFileName
			}
			if !uc && ed.Dirty {
				ed.Dirty = false
				return ErrFileModified
			}
			if fname == "" {
				fname = ed.Path
			}
			if err := ed.ReadFile(fname); err != nil {
				return err
			}
			log.Printf("Path: '%s'\n", ed.Path)
		}
		return nil

	case 'f':
		ed.nextToken()
		log.Printf("Token=%c\n", ed.token())
		if ed.token() == scanner.EOF {
			if ed.Path == "" {
				return ErrNoFileName
			}
			fmt.Fprintf(ed.err, "%s\n", ed.Path)
			return nil
		}
		ed.nextToken()
		var fname string = ed.scanString()
		log.Printf("Filename: '%s'\n", fname)
		if fname == "" {
			return ErrNoFileName
		}
		ed.Path = fname
		fmt.Fprintf(ed.err, "%s\n", ed.Path)
		return nil

	case 'g':
		return fmt.Errorf("TODO: g (global) not implemented")

	case 'G':
		return fmt.Errorf("TODO: G (interactive g) not implemented")

	case 'H':
		ed.printErrors = !ed.printErrors
		return nil

	case 'h':
		if ed.Error != nil {
			fmt.Fprintf(ed.err, "%s\n", ed.Error)
		}
		return nil

	case 'i':
		for {
			line, err := ed.ReadInsert()
			if err != nil {
				ed.setupSignals()
				goto end_insert
			}
			if line == "." {
				break
			}
			if len(ed.Lines) == ed.End {
				ed.Lines = append(ed.Lines, line)
				ed.End++
				continue
			}
			if ed.End-1 < 0 {
				return ErrInvalidAddress
			}
			ed.Lines = append(ed.Lines[:ed.End], ed.Lines[ed.End-1:]...)
			ed.Lines[ed.End-1] = line
			ed.End++
			ed.Dirty = true
		}
	end_insert:
		ed.Dot = ed.End
		ed.Start = ed.Dot
		ed.addr = ed.Dot
		return nil

	case 'j':
		if ed.End == ed.Start {
			ed.End++
		}
		if ed.End > len(ed.Lines) {
			return ErrInvalidAddress
		}
		var joined string = strings.Join(ed.Lines[ed.Start-1:ed.End], "")
		var result []string = append(append([]string{}, ed.Lines[:ed.Start-1]...), joined)
		ed.Lines = append(result, ed.Lines[ed.End:]...)
		ed.Dot = ed.Start
		ed.End = ed.Dot
		ed.addr = ed.Dot
		ed.Dirty = true
		return nil

	case 'k':
		ed.nextToken()
		var buf string = ed.scanString()
		switch len(buf) {
		case 1:
			break
		case 0:
			fallthrough
		default:
			return ErrInvalidCmdSuffix
		}
		var r rune = rune(buf[0])
		if !unicode.IsLower(r) {
			return ErrInvalidMark
		}
		log.Printf("Mark %c\n", r)
		var mark int = int(byte(buf[0])) - 'a'
		if mark >= len(ed.Mark) {
			return ErrInvalidMark
		}
		log.Printf("Mark %d is set to End (%d)\n", mark, ed.End)
		ed.Mark[int(mark)] = ed.End
		return nil

	case 'm':
		var err error
		var dst int
		ed.nextToken()
		log.Printf("Destination: %c\n", ed.token())
		var arg string = ed.scanString()
		dst, err = strconv.Atoi(arg)
		if err != nil {
			return ErrDestinationExpected
		}
		if dst < 0 || dst > len(ed.Lines) {
			// TODO: OpenBSD ed will evaluate the destination address,
			// so `24,26m-5` is actually a valid command
			return ErrDestinationExpected
		}
		log.Printf("Destination (arg): %d (%s)\n", dst, arg)
		lines := make([]string, ed.End-ed.Start+1)
		copy(lines, ed.Lines[ed.Start-1:ed.End+1])
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		ed.Lines = append(ed.Lines[:dst], append(lines, ed.Lines[dst:]...)...)
		ed.Dot = dst + len(lines)
		ed.End = ed.Dot
		ed.Start = ed.Dot
		return err

	case 'l':
		fallthrough
	case 'n':
		fallthrough
	case 'p':
		for i := ed.Start - 1; i < ed.End; i++ {
			if i < 0 {
				continue
			}
			switch ed.token() {
			case 'l':
				var q string = strconv.QuoteToASCII(ed.Lines[i])
				fmt.Fprintf(ed.out, "%s$\n", q[1:len(q)-1])
			case 'n':
				fmt.Fprintf(ed.out, "%d\t%s\n", i+1, ed.Lines[i])
			case 'p':
				fmt.Fprintf(ed.out, "%s\n", ed.Lines[i])
			}
		}
		return nil

	case 'P':
		if ed.Prompt == 0 {
			ed.Prompt = defaultPrompt
		} else {
			ed.Prompt = 0
		}
		return nil

	case 'q':
		fallthrough
	case 'Q':
		if ed.token() == 'q' && ed.Dirty {
			ed.Dirty = false
			return ErrFileModified
		}
		os.Exit(0)
		return nil

	case 'r':
		return fmt.Errorf("TODO: r (read) not implemented")

	case 's':
		return fmt.Errorf("TODO: s (substitute) not implemented")

	case 't':
		return fmt.Errorf("TODO: t (transfer) not implemented")

	case 'u':
		return fmt.Errorf("TODO: u (undo) not implemented")

	case 'v':
		return fmt.Errorf("TODO: v (inverse g) not implemented")

	case 'V':
		return fmt.Errorf("TODO: V (inverse V) not implemented")

	case 'W':
		fallthrough
	case 'w':
		var quit bool
		var r rune = ed.token()
		var full bool = (ed.s.Pos().Offset == 1)
		ed.nextToken()
		if r == 'w' {
			log.Printf("Write\n")
			if ed.token() == 'q' {
				ed.nextToken()
				quit = true
				log.Printf("Quit=%t\n", quit)
			}
		} else {
			log.Printf("Write (Append)\n")
		}
		if ed.token() == ' ' {
			ed.nextToken()
		}
		var fname string = ed.scanString()
		if fname == "" && ed.Path == "" {
			return ErrNoFileName
		}
		log.Printf("ed.Path: '%s'\n", ed.Path)
		if fname == "" {
			fname = ed.Path
		}
		var s int = ed.Start
		var e int = ed.End
		if full {
			log.Printf("Writing the whole file\n")
			s = 1
			e = len(ed.Lines)
		}
		var err error
		if r == 'w' {
			err = ed.WriteFile(s, e, fname)
		} else {
			err = ed.AppendFile(s, e, fname)
		}
		if quit {
			os.Exit(0)
		}
		return err

	case 'z':
		return fmt.Errorf("TODO: z (scroll) not implemented")

	case '=':
		fmt.Fprintf(ed.out, "%d\n", len(ed.Lines))
		return nil

	case '!':
		ed.nextToken()
		ed.consumeWhitespace()
		var buf string
		if ed.token() == scanner.EOF {
			if ed.Cmd != "" {
				buf = ed.Cmd
			} else {
				return ErrNoCmd
			}
		} else {
			buf = ed.scanString()
		}
		log.Printf("Command (unparsed): '%s'\n", buf)
		output, err := ed.Shell(buf)
		if err != nil {
			return err
		}
		for i := range output {
			fmt.Fprintf(ed.err, "%s\n", output[i])
		}
		fmt.Fprintln(ed.err, "!")
		return nil

	case 0:
		fallthrough
	case scanner.EOF:
		if ed.End-1 < 0 || ed.End-1 > len(ed.Lines) {
			return ErrInvalidAddress
		}
		fmt.Fprintf(ed.out, "%s\n", ed.Lines[ed.End-1])
		return nil
	default:
		return ErrUnknownCmd
	}
}
