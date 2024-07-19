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
	defer os.Remove(fname)
	setupMemoryFile(ted, buf)
	ted.modified = true
	ted.printErrors = true
	go func() {
		ted.sighupch <- syscall.SIGHUP
	}()
	select {
	case <-time.After(500 * time.Millisecond):
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
		dur      = 2 * time.Second
		timeout  = time.After(dur)
	)
	ted.in = strings.NewReader("a\nABCDEF")
	ted.printErrors = true
	go func() {
		ted.sigintch <- syscall.SIGINT
	}()
	if err := ted.Do(); err != ErrInterrupt {
		t.Fatal(err)
	}
	select {
	case <-time.After(500 * time.Millisecond):
		if reflect.DeepEqual(ted.lines, expected) {
			t.Fatalf("expected buffer to be %q, got %q", expected, ted.lines)
		}
		if ted.error != ErrInterrupt {
			t.Fatalf("expected error %q, got %q", ErrInterrupt, ted.error)
		}
	case <-timeout:
		t.Fatalf("timed out after %s", dur)
	}
}
