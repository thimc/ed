package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/scanner"
)

var (
	debugFlag    = flag.Bool("d", false, "toggles debug information")
	promptFlag   = flag.Bool("p", false, "toggles the prompt")
	suppressFlag = flag.Bool("s", false, "suppress diagnostics")
)

type Editor struct {
	Path  string
	Dirty bool
	Lines []string
	Mark  ['z' - 'a']int

	Dot   int
	Start int
	End   int

	input []byte

	Search string
	Error  error
	Prompt rune
	Cmd    string

	printErrors bool

	s   *scanner.Scanner
	tok *rune
}

var (
	ed Editor

	DefaultError  = errors.New("?")
	DefaultPrompt = '*'
)

func (ed *Editor) readFile(path string) (int64, error) {
	var siz int64
	file, err := os.Open(path)
	if err != nil {
		return siz, err
	}
	stat, err := file.Stat()
	if err != nil {
		return siz, err
	}
	siz = stat.Size()
	s := bufio.NewScanner(file)
	for s.Scan() {
		ed.Lines = append(ed.Lines, s.Text())
	}
	if err := s.Err(); err != nil {
		return siz, err
	}
	ed.Path = path
	ed.Dot = len(ed.Lines)
	ed.Start = 1
	ed.End = ed.Dot
	return siz, nil
}

func readStdin(prompt rune) ([]byte, error) {
	var err error
	var line []byte
	if prompt != 0 {
		fmt.Fprintf(os.Stdout, "%c", prompt)
	}
	r := bufio.NewReader(os.Stdin)
	line, err = r.ReadBytes('\n')
	if err != nil {
		return line, err
	}
	return line[:len(line)-1], err
}

func main() {
	flag.Parse()
	if !(*debugFlag) {
		log.SetOutput(ioutil.Discard)
	}

	if *promptFlag {
		ed.Prompt = DefaultPrompt
	}

	var args []string = flag.Args()
	if len(args) == 1 {
		siz, err := ed.readFile(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			ed.Error = errors.New("cannot open input file")
		} else {
			log.Printf("Open file %s\n", args[0])
			fmt.Fprintf(os.Stderr, "%d\n", siz)
		}
	}

	for {
		var err error
		ed.input, err = readStdin(ed.Prompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		log.Printf("Read input: '%s'\n", ed.input)
		if err := ed.DoRange(); err != nil {
			ed.Error = err
			fmt.Fprintf(os.Stderr, "%s\n", err)
			if ed.printErrors {
				fmt.Fprintf(os.Stderr, "%s\n", DefaultError)
			}
			continue
		}
		if err := ed.DoCommand(); err != nil {
			ed.Error = err
			fmt.Fprintf(os.Stderr, "%s\n", err)
			if ed.printErrors {
				fmt.Fprintf(os.Stderr, "%s\n", DefaultError)
			}
			continue
		}
	}
}
