// Command ed implements ed, the standard unix text editor.
// Originally created by Ken Thompson in the 1970s.
//
// Usage:
//
//	ed [-] [-s] [-p string] [file]
//
// ed is a line-oriented text editor that operates on a file one line at a time.
// It's designed for editing small to medium-sized files, and its simplicity makes
// it a great choice for quick edits or when you need to edit files in a script.
//
// Basic commands:
//
//	s/old/new/g : Substitutes all occurences of old with new on the current line
//	a           : Append lines at [dot] until the entered line only consists of a "."
//	c           : Change lines selected by the range until the entered line only consists of a "."
//	d           : Delete the current line
//	p           : Print the current line
//	,p          : Prints the entire buffer
//	,n          : Prints the entire buffer but with line numbers
//	q           : Quit ed
//
// For more information, refer to the OpenBSD man page: https://man.openbsd.org/ed.1
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

var (
	Prompt = flag.String("p", "", "user prompt")
	Silent = flag.Bool("s", false, "suppress diagnostics")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [-] [-s] [-p string] [file]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	flag.Parse()
	opts := []Option{WithStdin(os.Stdin), WithPrompt(*Prompt), WithSilent(*Silent)}
	if flag.NArg() > 0 {
		arg := flag.Args()[0]
		if arg == "-" {
			*Silent = true
		} else {
			opts = append(opts, WithFile(arg))
		}
	}
	NewEditor(opts...).Run()
}
