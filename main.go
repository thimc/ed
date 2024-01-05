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

	DefaultError  = errors.New("?") // descriptive error message, don't you think?
	DefaultPrompt = '*'
)

func (ed *Editor) readFile(path string) error {
	var siz int64
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open input file")
	}
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot open input file")
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
	ed.Start = 1
	ed.End = ed.Dot
	fmt.Fprintf(os.Stderr, "%d\n", siz)
	return nil
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
		if err := ed.readFile(args[0]); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		} else {
			log.Printf("Open file %s\n", args[0])
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
