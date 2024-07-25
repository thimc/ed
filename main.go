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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	// prompt sets the user prompt and implicitly enables the prompt option.
	prompt = flag.String("p", "", "user prompt")
	// silent surpresses diagnostics and should be if ed is used in scripts.
	silent = flag.Bool("s", false, "suppress diagnostics")
)

func main() {
	flag.Parse()
	fi, _ := os.Stdin.Stat()
	var (
		args     = flag.Args()
		terminal = (fi.Mode() & os.ModeCharDevice) > 0
	)
	for n, arg := range os.Args {
		if arg == "-" {
			*silent = true
			args = append(args[:n-1], args[n:]...)
		}
	}
	var options = []OptionFunc{WithPrompt(*prompt), WithSilent(*silent), WithScripted(!terminal)}
	if len(args) == 1 {
		// TODO(thimc): Support "binary mode" which replaces all
		// instances of the NULL or nil token with a newline.
		options = append(options, WithFile(args[0]))
	}
	for ed := New(options...); ; {
		if err := ed.Do(); err != nil {
			if errors.Is(err, io.EOF) && !terminal {
				break
			}
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
