package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/scanner"
)

var (
	promptFlag   = flag.Bool("p", false, "toggles the prompt")
	suppressFlag = flag.Bool("s", false, "suppress diagnostics")
)

type Editor struct {
	Path  string
	Dirty bool
	Lines []string

	Dot   int
	Start int
	End   int

	input []byte

	Search string
	Error  error
	Prompt rune

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
	// log.SetOutput(ioutil.Discard)

	if *promptFlag {
		ed.Prompt = DefaultPrompt
	}

	// var args []string = os.Args[1:]
	// if len(args) == 1 {
	// 	err := ed.readFile(args[0])
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	siz, _ := ed.readFile("test.txt")
	fmt.Println(siz)
	test := false

	for {

		if test {

			tests := [][]byte{
				[]byte("4"),
				[]byte("4,+4p"),
				[]byte("-5,-3p"),
				[]byte("^2,^1p"),
				[]byte(";p"),
			}
			for i := range tests {
				ed.input = tests[i]
				fmt.Printf("Test: \"%s\"\n", string(ed.input))
				if err := ed.DoRange(); err != nil {
					if !ed.printErrors {
						err = DefaultError
					}
					fmt.Fprintf(os.Stderr, "%s\n", err)
					continue
				}
				if err := ed.DoCommand(); err != nil {
					if !ed.printErrors {
						err = DefaultError
					}
					fmt.Fprintf(os.Stderr, "%s\n", err)
					continue
				}
			}
			break
		} else {

			var err error
			ed.input, err = readStdin(ed.Prompt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err)
				os.Exit(1)
			}
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
}
