// Command ed implements ed, the standard unix text editor.
// Originally created by Ken Thompson in the 1970s.
//
// Usage:
//
//	ed [-s] [-p prompt] [file]
//
// Ed is a line-oriented text editor that operates on a file one line at a time.
// It's designed for editing small to medium-sized files, and its simplicity makes
// it a great choice for quick edits or when you need to edit files in a script.
//
// Basic Commands:
//
//	s/old/new/g : Substitutes all occurences of old with new on the current line
//	d  : Delete the current line
//	p  : Print the current line
//	,p : Prints the entire buffer
//	q  : Quit ed
//
// For more information, see the ed manual page or the OpenBSD
// documentation: https://man.openbsd.org/ed.1
package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	// Prompt sets the user Prompt and implicitly enables the Prompt option.
	Prompt = flag.String("p", "", "user prompt")
	// Scripted surpresses diagnostics and should be if ed is used in scripts.
	Scripted = flag.Bool("s", false, "suppress diagnostics")
)

func main() {
	flag.Parse()
	var (
		args    = flag.Args()
		options = []OptionFunc{WithPrompt(*Prompt), WithScripted(*Scripted)}
	)
	// TODO(thimc): Add support for the (deprecated) `-` flag
	// as an alternative way to enable scripted mode.
	if len(flag.Args()) == 1 {
		options = append(options, WithFile(args[0]))
	}
	for ed := New(options...); ; {
		if err := ed.Do(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
