//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"fmt"
	"os/signal"
	"syscall"
)

func (ed *Editor) handleSignals() {
	signal.Notify(ed.sigch, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	for sig := range ed.sigch {
		switch sig {
		case syscall.SIGINT:
			ed.err = ErrInterrupt
			fmt.Fprintf(ed.stdout, "\n%s\n", ErrDefault)
			// TODO(thimc): SIGINT: Return to command mode on interrupt.
		case syscall.SIGHUP:
			if ed.file.dirty && len(ed.file.lines) > 0 {
				ed.file.write(DefaultHangupFile, 'w', 1, len(ed.file.lines))
			}
		case syscall.SIGQUIT:
			// ignore
		}
		if ed.verbose {
			fmt.Fprintln(ed.stderr, ed.err)
		}
	}
}
