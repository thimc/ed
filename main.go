package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	debugFlag    = flag.Bool("d", false, "toggles debug information")
	promptFlag   = flag.String("p", "", "prompt")
	suppressFlag = flag.Bool("s", false, "suppress diagnostics")
)

func (ed *Editor) printError(err error) bool {
	if err != nil {
		ed.Error = err
		if ed.printErrors {
			fmt.Fprintf(ed.err, "%s\n", ed.Error)
		} else {
			fmt.Fprintf(ed.err, "%s\n", ErrDefault)
		}
		return true
	}
	return false
}

func main() {
	var ed *Editor
	flag.Parse()
	ed = NewEditor(os.Stdin, os.Stdout, os.Stderr)
	ed.Prompt = *promptFlag
	if *promptFlag != "" {
		ed.showPrompt = true
	}
	ed.Silent = *suppressFlag
	var args []string = flag.Args()
	if len(args) == 1 {
		var err error
		ed.Lines, err = ed.ReadFile(args[0], true, true)
		if err != nil {
			fmt.Fprintf(ed.err, "%s\n", err)
		}
		ed.Path = args[0]
	}
	for {
		if err := ed.ReadInput(os.Stdin); err != nil {
			break
		}
		if ed.printError(ed.DoRange()) {
			continue
		}
		if ed.printError(ed.DoCommand()) {
			continue
		}
	}
}
