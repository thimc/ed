package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestSignalSIGHUP tests the SIGHUP functionality which should write a
// "dirty" buffer to disk.
func TestSignalSIGHUP(t *testing.T) {
	var (
		b        bytes.Buffer
		ted      = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(&b))
		buf      = []string{"A", "B", "C", "D", "E", "F"}
		fname    = DefaultHangupFile
		dur      = 2 * time.Second
		timeout  = time.After(dur)
		expected = fmt.Sprintf("%d\n", len(buf)*2)
	)
	ted.removeDummyFile(fname)
	defer ted.removeDummyFile(fname)
	ted.setupTestFile(buf)
	ted.dirty = true
	ted.printErrors = true
	go func() {
		ted.sighupch <- syscall.SIGHUP
	}()
	select {
	case <-time.After(100 * time.Millisecond):
		if _, err := os.Stat(fname); err != nil {
			t.Fatalf("expected file %q to exist, got error %q", fname, err)
		}
		if b.String() != expected {
			t.Fatalf("expected output %q, got %q", expected, b.String())
		}
	case <-timeout:
		t.Fatalf("timed out after %s", dur)
	}
}

// TestSignalSIGINT tests the SIGINT functionality which break out of
// `append`, `insert` or `change` mode.
func TestSignalSIGINT(t *testing.T) {
	var (
		ted      = New(WithStdin(nil), WithStdout(io.Discard), WithStderr(io.Discard))
		expected = []string{"A", "B"}
	)
	ted.in = strings.NewReader("a\nA\nB\nC")
	go func() {
		ted.sigintch <- syscall.SIGINT
	}()
	if reflect.DeepEqual(ted.Lines, expected) {
		t.Fatalf("expected buffer to be %q, got %q", expected, ted.Lines)
	}
	if ted.error != ErrInterrupt {
		t.Fatalf("expected error %q, got %q", ErrInterrupt, ted.error)
	}
}
