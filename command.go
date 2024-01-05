package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/scanner"
)

func (ed *Editor) Shell(command string) ([]string, error) {
	var output []string
	cmd := exec.Command("/bin/sh", "-c", command)
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
	return output, err
}

func (ed *Editor) ReadInsert() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	return reader.ReadString('\n')
}

func (ed *Editor) DoCommand() error {
	var err error
	var s scanner.Scanner = *ed.s
	var tok rune = *ed.tok

	log.Printf("Cmd='%c'\n", tok)
	switch tok {
	case 'a':
		for {
			line, _ := ed.ReadInsert()
			line = line[:len(line)-1]
			if line == "." {
				break
			}
			if tok == 'a' {
				ed.Lines = append(ed.Lines, "")
				copy(ed.Lines[ed.Dot:], ed.Lines[ed.Dot:])
			}
			ed.Lines[ed.Dot] = line
			ed.Dot++
			ed.Dirty = true
		}
	case 'c':
		ed.Dirty = true
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.Dot:]...)
		ed.Dot = ed.Start - 1
		for {
			line, _ := ed.ReadInsert()
			line = line[:len(line)-1]
			if line == "." {
				break
			}
			ed.Lines = append(ed.Lines[:ed.Dot+1], ed.Lines[ed.Dot:]...)
			ed.Lines[ed.Dot] = line
			ed.Dot++
		}
	case 'd':
		ed.Dirty = true
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.Dot:]...)
		ed.Dot = ed.Start
	case 'e':
		tok = s.Scan()
		if tok == '!' {
			return fmt.Errorf("not implemented") // TODO edit the standard output of command
		}
		return fmt.Errorf("not implemented") // TODO edit
	case 'E':
		return fmt.Errorf("not implemented") // TODO edit (unconditionally)
	case 'f':
		tok = s.Scan()
		log.Printf("Token=%c\n", tok)
		if tok == scanner.EOF {
			if ed.Path == "" {
				return fmt.Errorf("no current filename")
			}
			fmt.Fprintf(os.Stderr, "%s\n", ed.Path)
		}
		var filename string
		for tok != scanner.EOF {
			filename += string(tok)
			tok = s.Scan()
		}
		if filename == "" {
			return fmt.Errorf("invalid filename")
		}
		ed.Path = filename
		fmt.Fprintf(os.Stderr, "%s\n", ed.Path)
	case 'g':
		return fmt.Errorf("not implemented") // TODO
	case 'G':
		return fmt.Errorf("not implemented") // TODO
	case 'H':
		ed.printErrors = !ed.printErrors
	case 'h':
		fmt.Fprintf(os.Stderr, "%s\n", ed.Error)
	case 'i':
		for {
			line, _ := ed.ReadInsert()
			line = line[:len(line)-1]
			if line == "." {
				break
			}
			ed.Lines = append(ed.Lines[:ed.Dot], ed.Lines[ed.Dot-1:]...)
			ed.Lines[ed.Dot-1] = line
			ed.Dot++
		}
	case 'j':
		var joined string = strings.Join(ed.Lines[ed.Start-1:ed.Dot], "")
		var result []string = append(append([]string{}, ed.Lines[:ed.Start-1]...), joined)
		ed.Lines = append(result, ed.Lines[ed.End:]...)
		ed.Dot = ed.Start
		ed.Dirty = true
	case 'k':
		tok = s.Scan()
		var mark byte = byte(tok) - 'a'
		if tok == scanner.EOF || s.Peek() != scanner.EOF || int(mark) >= len(ed.Mark) {
			return fmt.Errorf("invalid command suffix")
		}
		log.Printf("Mark %d is set to Dot (%d)\n", int(mark), ed.Dot)
		ed.Mark[int(mark)] = ed.Dot
	case 'm':
		var arg string
		var dst int
		tok = s.Scan()
		log.Printf("Destination: %c\n", tok)
		for tok != scanner.EOF {
			arg += string(tok)
			tok = s.Scan()
		}
		dst, err = strconv.Atoi(arg)
		if err != nil {
			return fmt.Errorf("destination expected")
		}
		log.Printf("Destination (arg): %d (%s)\n", dst, arg)
		lines := make([]string, ed.End-ed.Start+1)
		copy(lines, ed.Lines[ed.Start-1:ed.End+1])
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.End:]...)
		ed.Lines = append(ed.Lines[:dst], append(lines, ed.Lines[dst:]...)...)
		ed.Dot = dst + len(lines)
	case 'l':
		fallthrough
	case 'n':
		fallthrough
	case 'p':
		for i := ed.Start - 1; i+1 <= ed.End; i++ {
			if i < 0 {
				continue
			}
			switch tok {
			case 'l':
				var q string = strconv.QuoteToASCII(ed.Lines[i])
				fmt.Fprintf(os.Stdout, "%s$\n", q[1:len(q)-1])
			case 'n':
				fmt.Fprintf(os.Stdout, "%d\t%s\n", i+1, ed.Lines[i])
			case 'p':
				fmt.Fprintf(os.Stdout, "%s\n", ed.Lines[i])
			}
		}
		ed.Dot = ed.End
	case 'P':
		if ed.Prompt == 0 {
			ed.Prompt = DefaultPrompt
		} else {
			ed.Prompt = 0
		}
	case 'q':
		fallthrough
	case 'Q':
		if tok == 'q' && ed.Dirty {
			ed.Dirty = false
			return fmt.Errorf("warning: file modified")
		}
		os.Exit(0)
	case 'r':
		return fmt.Errorf("not implemented") // TODO read
	case 's':
		return fmt.Errorf("not implemented") // TODO substitute
	case 't':
		return fmt.Errorf("not implemented") // TODO transfer
	case 'u':
		return fmt.Errorf("not implemented") // TODO undo
	case 'v':
		return fmt.Errorf("not implemented") // TODO
	case 'V':
		return fmt.Errorf("not implemented") // TODO
	case 'w':
		if s.Peek() == 'q' {
			return fmt.Errorf("not implemented") // TODO write quit
		}
		return fmt.Errorf("not implemented") // TODO write
	case 'W':
		return fmt.Errorf("not implemented") // TODO write
	case 'z':
		return fmt.Errorf("not implemented") // TODO scroll
	case '=':
		fmt.Fprintf(os.Stdout, "%d\n", len(ed.Lines))
	case '!':
		tok = s.Scan()
		var buf string
		if tok == scanner.EOF {
			buf = ed.Cmd
		} else {
			for tok != scanner.EOF {
				buf += string(tok)
				tok = s.Scan()
			}
		}
		log.Printf("Command (unparsed): '%s'\n", buf)
		ed.Cmd = buf
		var cs scanner.Scanner
		cs.Init(strings.NewReader(buf))
		cs.Mode = scanner.ScanChars
		cs.Whitespace ^= scanner.GoWhitespace
		var cmd string
		var ctok rune = cs.Scan()
		for ctok != scanner.EOF {
			cmd += string(ctok)
			if ctok != '\\' && cs.Peek() == '%' {
				ctok = cs.Scan()
				log.Printf("Replacing %% with '%s'\n", ed.Path)
				cmd += ed.Path
			}
			ctok = cs.Scan()
		}
		output, err := ed.Shell(cmd)
		if err != nil {
			return err
		}
		for i := range output {
			fmt.Fprintf(os.Stderr, "%s\n", output[i])
		}
		fmt.Fprintln(os.Stderr, "!")
	default:
		return fmt.Errorf("unknown command")
	}
	return err
}
