package main

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

var dummyFile = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
}

// setupMemoryFile initializes a in-memory buffer
func setupMemoryFile(ed *Editor, buf []string) {
	ed.Lines = buf
	ed.path = "test"
	ed.dot = len(buf)
	ed.start = ed.dot
	ed.end = ed.dot
	ed.printErrors = true
}

// createDummyFile creates a dummy file `fname` containing `dummyFile`.
func createDummyFile(fname string) error {
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(strings.Join(dummyFile, "\n"))
	return err
}

func TestParser(t *testing.T) {
	var (
		ted  = New(WithStdout(io.Discard), WithStderr(io.Discard))
		last = len(dummyFile)
	)
	tests := []struct {
		cmd    string
		init   position
		expect position
	}{
		{
			cmd: "8",
			init: position{
				start: last,
				end:   last,
				dot:   last,
			},
			expect: position{
				start: 8,
				end:   8,
				dot:   last,
				addrc: 1,
			},
		},
		{
			cmd: "1,5",
			init: position{
				start: last,
				end:   last,
				dot:   last,
			},
			expect: position{
				start: 1,
				end:   5,
				dot:   last,
				addrc: 2,
			},
		},
		{
			cmd: "+",
			init: position{
				start: 2,
				end:   2,
				dot:   2,
			},
			expect: position{
				start: 3,
				end:   3,
				dot:   2,
				addrc: 1,
			},
		},
		{
			cmd: "-",
			init: position{
				start: 3,
				end:   3,
				dot:   3,
			},
			expect: position{
				start: 2,
				end:   2,
				dot:   3,
				addrc: 1,
			},
		},
		{
			cmd: "^",
			init: position{
				start: 3,
				end:   3,
				dot:   3,
			},
			expect: position{
				start: 2,
				end:   2,
				dot:   3,
				addrc: 1,
			},
		},
		{
			cmd: ".,+5",
			init: position{
				start: 4,
				end:   4,
				dot:   4,
			},
			expect: position{
				start: 4,
				end:   9,
				dot:   4,
				addrc: 2,
			},
		},
		{
			cmd: "-2,+5",
			init: position{
				start: 5,
				end:   5,
				dot:   5,
			},
			expect: position{
				start: 3,
				end:   10,
				dot:   5,
				addrc: 2,
			},
		},
		{
			cmd: ",",
			init: position{
				start: 5,
				end:   5,
				dot:   5,
			},
			expect: position{
				start: 1,
				end:   last,
				dot:   5,
				addrc: 2,
			},
		},
		{
			cmd: "6,%",
			init: position{
				start: 5,
				end:   5,
				dot:   5,
			},
			expect: position{
				start: 1,
				end:   last,
				dot:   5,
				addrc: 2,
			},
		},
		{
			cmd: "3;",
			init: position{
				start: 5,
				end:   5,
				dot:   5,
			},
			expect: position{
				start: 3,
				end:   3,
				dot:   5,
				addrc: 1,
			},
		},
	}
	setupMemoryFile(ted, dummyFile)
	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			ted.in = strings.NewReader(tt.cmd)
			ted.start = tt.init.start
			ted.end = tt.init.end
			ted.dot = tt.init.dot
			ted.addrc = tt.init.addrc

			ted.Tokenizer = NewTokenizer(ted.in)
			ted.token()

			if err := ted.parse(); err != nil {
				t.Fatal(err)
			}
			got := position{
				start: ted.start,
				end:   ted.end,
				dot:   ted.dot,
				addrc: ted.addrc,
			}
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("expected %+v, got %+v", tt.expect, got)
			}
		})
	}
}
