//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func (ed *Editor) handleSignals() {
	signal.Notify(ed.sigch, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	for sig := range ed.sigch {
		switch sig {
		case os.Interrupt:
			ed.err = ErrInterrupt
			fmt.Fprintf(ed.stdout, "\n%s\n", ErrDefault)
			// TODO: Return to command mode.
		case syscall.SIGHUP:
			// TODO: If the current buffer has changed since it was last written, ed
			// attempts to write the buffer to the file ed.hup.  Nothing is
			// written to the currently remembered file, and ed exits.
		case syscall.SIGQUIT:
			// ignore
		}
		if ed.verbose {
			fmt.Fprintln(ed.stderr, ed.err)
		}
	}
}
