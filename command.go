package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/scanner"
)

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
		ed.Dirty = true
		for {
			line, _ := ed.ReadInsert()
			line = line[:len(line)-1]
			if line == "." {
				break
			}
			ed.Lines = append(ed.Lines, "")
			copy(ed.Lines[ed.Dot:], ed.Lines[ed.Dot:])
			ed.Lines[ed.Dot] = line
			ed.Dot++
		}
	case 'c':
		ed.Dirty = true
		return fmt.Errorf("not implemented") // TODO change
	case 'd':
		ed.Dirty = true
		ed.Lines = append(ed.Lines[:ed.Start-1], ed.Lines[ed.Dot:]...)
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
			ed.Lines = append([]string{line}, ed.Lines...)
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
		ed.Mark[int(mark)] = ed.Dot
	case 'm':
		return fmt.Errorf("not implemented") // TODO move lines
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
		return fmt.Errorf("not implemented") // TODO execute
	default:
		return fmt.Errorf("unknown command")
	}
	return err
}
