package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	debugFlag    = flag.Bool("d", false, "toggles debug information")
	promptFlag   = flag.Bool("p", false, "toggles the prompt")
	suppressFlag = flag.Bool("s", false, "suppress diagnostics")
)
var ed *Editor

func printError(err error) bool {
	if err != nil {
		ed.Error = err
		fmt.Fprintf(os.Stderr, "%s\n", err)
		if ed.printErrors {
			fmt.Fprintf(os.Stderr, "%s\n", ErrDefault)
		}
		return true
	}
	return false
}

func main() {
	flag.Parse()
	ed = NewEditor(os.Stdin, os.Stdout, os.Stderr)
	if !*debugFlag {
	}
	if *promptFlag {
		ed.Prompt = defaultPrompt
	}
	ed.Silent = *suppressFlag
	var args []string = flag.Args()
	if len(args) == 1 {
		var err error
		ed.Lines, err = ed.ReadFile(args[0])
		if !printError(err) {
			ed.Path = args[0]
		}
	}
	for {
		if ed.Prompt != 0 {
			fmt.Fprintf(os.Stderr, "%c", ed.Prompt)
		}
		if err := ed.ReadInput(os.Stdin); err != nil {
			panic(err)
		}
		if printError(ed.DoRange()) {
			continue
		}
		if printError(ed.DoCommand()) {
			continue
		}
	}
}
