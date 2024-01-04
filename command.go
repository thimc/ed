package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
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

	// log.Printf("Cmd=%c\n", tok)
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
			ed.Dot++
			ed.Lines[ed.Dot] = line
		}
	case 'c':
		ed.Dirty = true
		return fmt.Errorf("not implemented") // TODO change
	case 'd':
		ed.Dirty = true
		return fmt.Errorf("not implemented") // TODO delete
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
		return fmt.Errorf("not implemented") // TODO insert
	case 'j':
		return fmt.Errorf("not implemented") // TODO join lines
	case 'k':
		return fmt.Errorf("not implemented") // TODO mark
	case 'l':
		return fmt.Errorf("not implemented") // TODO print lines unambiguously
	case 'm':
		return fmt.Errorf("not implemented") // TODO move lines
	case 'n':
		fallthrough
	case 'p':
		for i := ed.Start - 1; i < ed.End; i++ {
			if tok == 'n' {
				fmt.Fprintf(os.Stdout, "%d\t%s\n", i+1, ed.Lines[i])
			} else {
				fmt.Fprintf(os.Stdout, "%s\n", ed.Lines[i])
			}
			ed.Dot = i
		}
	case 'P':
		if ed.Prompt == 0 {
			ed.Prompt = DefaultPrompt
		} else {
			ed.Prompt = 0
		}
	case 'q':
		return fmt.Errorf("not implemented") // TODO quit
	case 'Q':
		// log.Println("Quit ed unconditionally")
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
		fmt.Fprintf(os.Stdout, "%d\n", ed.Dot+1)
	case '!':
		return fmt.Errorf("not implemented") // TODO execute
	}
	return err
}
