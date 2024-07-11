// Command ed implements the OpenBSD variant of ed, the standard text
// editor in Unix. Originally created by Ken Thompson in the 1970s.
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	prompt = flag.String("p", "", "user prompt")
	silent = flag.Bool("s", false, "suppress diagnostics")
)

func main() {
	flag.Parse()
	var (
		args    = flag.Args()
		options = []OptionFunc{WithPrompt(*prompt), WithSilent(*silent)}
	)
	if len(args) == 1 {
		options = append(options, WithFile(args[0]))
	}
	ed := New(options...)
	for {
		if err := ed.Do(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
