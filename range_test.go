package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"testing"
)

// TestRangePass() runs tests on the range parser.  It expects there
// to be a file in the current working directory called "file". All
// the tests in this function are done in the same editing session
// and all are expected to pass.
// Some of the test values have special meaning:
//
//	 0: length of the buffer
//	-1: skip test
func TestRangePass(t *testing.T) {
	var ted *Editor = NewEditor(io.Discard, io.Discard)
	ted.setupTestFile()
	log.SetOutput(io.Discard)

	tests := []struct {
		input           []byte
		expectStart     int
		expectEnd       int
		expectDot       int
		expectAddr      int
		expectAddrCount int
	}{
		{
			input:           []byte("8"),
			expectStart:     8,
			expectEnd:       8,
			expectDot:       8,
			expectAddr:      8,
			expectAddrCount: 1,
		},
		{
			input:           []byte("1,5"),
			expectStart:     1,
			expectEnd:       5,
			expectDot:       5,
			expectAddr:      5,
			expectAddrCount: 2,
		},
		{
			input:           []byte("3"),
			expectStart:     3,
			expectEnd:       3,
			expectDot:       3,
			expectAddr:      3,
			expectAddrCount: 1,
		},
		{
			input:           []byte(".,+5"),
			expectStart:     3,
			expectEnd:       8,
			expectDot:       8,
			expectAddr:      8,
			expectAddrCount: 2,
		},
		{
			input:           []byte(""),
			expectStart:     9,
			expectEnd:       9,
			expectDot:       9,
			expectAddr:      8,
			expectAddrCount: 0,
		},
		{
			input:           []byte("-3,+5"),
			expectStart:     6,
			expectEnd:       14,
			expectDot:       14,
			expectAddr:      14,
			expectAddrCount: 2,
		},
		{
			input:           []byte("-2,+2p"),
			expectStart:     12,
			expectEnd:       16,
			expectDot:       16,
			expectAddr:      16,
			expectAddrCount: 2,
		},
		{
			input:           []byte("="),
			expectStart:     16,
			expectEnd:       16,
			expectDot:       16,
			expectAddr:      16,
			expectAddrCount: 1,
		},
		{
			input:           []byte("+"),
			expectStart:     17,
			expectEnd:       17,
			expectDot:       17,
			expectAddr:      17,
			expectAddrCount: 1,
		},
		{
			input:           []byte(",n"),
			expectStart:     1,
			expectEnd:       0,
			expectDot:       0,
			expectAddr:      0,
			expectAddrCount: 2,
		},
		{
			input:           []byte("3;"),
			expectStart:     3,
			expectEnd:       0,
			expectDot:       0,
			expectAddr:      0,
			expectAddrCount: 2,
		},
		{
			input:           []byte("%"),
			expectStart:     1,
			expectEnd:       0,
			expectDot:       0,
			expectAddr:      0,
			expectAddrCount: 2,
		},
		{
			input:           []byte("-1"),
			expectStart:     -1,
			expectEnd:       -1,
			expectDot:       -1,
			expectAddr:      -1,
			expectAddrCount: -1,
		},
		{
			input:           []byte("1,5p"),
			expectStart:     1,
			expectEnd:       5,
			expectDot:       5,
			expectAddr:      5,
			expectAddrCount: 2,
		},
		{
			input:           []byte("1,5p"),
			expectStart:     1,
			expectEnd:       5,
			expectDot:       5,
			expectAddr:      5,
			expectAddrCount: 2,
		},
	}

	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.ReadInput(bytes.NewBuffer(test.input))

			if test.expectEnd == 0 {
				test.expectEnd = len(ted.Lines)
			}
			if test.expectDot == 0 {
				test.expectDot = len(ted.Lines)
			}
			if test.expectStart == 0 {
				test.expectStart = len(ted.Lines)
			}
			if test.expectAddr == 0 {
				test.expectAddr = len(ted.Lines)
			}

			if err := ted.DoRange(); err != nil {
				t.Fatalf("expected the range to be valid but failed: %s", err)
			}

			if err := ted.DoCommand(); err != nil {
				t.Fatalf("expected the command to be valid but failed: %s", err)
			}

			if test.expectStart != -1 && ted.Start != test.expectStart {
				t.Errorf("expected start position %d, got %d",
					test.expectStart, ted.Start)
			}
			if test.expectEnd != -1 && ted.End != test.expectEnd {
				t.Errorf("expected end position %d, got %d", test.expectEnd, ted.End)
			}
			if test.expectDot != -1 && ted.Dot != test.expectDot {
				t.Errorf("expected dot position %d, got %d", test.expectDot, ted.Dot)
			}
			if test.expectAddr != -1 && ted.addr != test.expectAddr {
				t.Errorf("expected internal address position %d, got %d",
					test.expectAddr, ted.addr)
			}
			if test.expectAddrCount != -1 && ted.addrcount != test.expectAddrCount {
				t.Errorf("expected internal address count %d, got %d",
					test.expectAddrCount, ted.addrcount)
			}

		})
	}
}

// TestRangeFail() runs tests on the range parser.  It expects there
// to be a file in the current working directory called "file". All
// the tests in this function are done in the same editing session
// and all are expected to fail except "ReadFile".
// Some of the test values have special meaning for convenient testing:
//
//	 0: length of the buffer
//	-1: skip test
func TestRangeFail(t *testing.T) {
	var ted Editor = *NewEditor(io.Discard, io.Discard)
	log.SetOutput(io.Discard)
	ted.setupTestFile()

	tests := []struct {
		input []byte
	}{
		{input: []byte("0")},
		{input: []byte(fmt.Sprintf("%d\n", len(ted.Lines)*10))},
		{input: []byte("-2,+")},
	}

	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			ted.ReadInput(bytes.NewBuffer(test.input))
			if err := ted.DoRange(); err == nil {
				t.Fatal("expected the range to fail but is valid")
			}
			if err := ted.DoCommand(); err == nil {
				t.Fatal("expected the command to fail but is valid")
			}
		})
	}
}

func (ed *Editor) setupTestFile() {
	ed.Lines = []string{
		"01AAAAAAAAAAAAAAAAAAAAAAAAA",
		"02BBBBBBBBBBBBBBBBBBBBBBBBB",
		"03CCCCCCCCCCCCCCCCCCCCCCCCC",
		"04DDDDDDDDDDDDDDDDDDDDDDDDD",
		"05EEEEEEEEEEEEEEEEEEEEEEEEE",
		"06FFFFFFFFFFFFFFFFFFFFFFFFF",
		"07GGGGGGGGGGGGGGGGGGGGGGGGG",
		"08HHHHHHHHHHHHHHHHHHHHHHHHH",
		"09IIIIIIIIIIIIIIIIIIIIIIIII",
		"10JJJJJJJJJJJJJJJJJJJJJJJJJ",
		"11KKKKKKKKKKKKKKKKKKKKKKKKK",
		"12LLLLLLLLLLLLLLLLLLLLLLLLL",
		"13MMMMMMMMMMMMMMMMMMMMMMMMM",
		"14NNNNNNNNNNNNNNNNNNNNNNNNN",
		"15OOOOOOOOOOOOOOOOOOOOOOOOO",
		"16PPPPPPPPPPPPPPPPPPPPPPPPP",
		"17QQQQQQQQQQQQQQQQQQQQQQQQQ",
		"18RRRRRRRRRRRRRRRRRRRRRRRRR",
		"19SSSSSSSSSSSSSSSSSSSSSSSSS",
		"20TTTTTTTTTTTTTTTTTTTTTTTTT",
		"21UUUUUUUUUUUUUUUUUUUUUUUUU",
		"22VVVVVVVVVVVVVVVVVVVVVVVVV",
		"23WWWWWWWWWWWWWWWWWWWWWWWWW",
		"24XXXXXXXXXXXXXXXXXXXXXXXXX",
		"25YYYYYYYYYYYYYYYYYYYYYYYYY",
		"26ZZZZZZZZZZZZZZZZZZZZZZZZZ",
	}
	ed.Path = "test"
	ed.Dot = len(ed.Lines)
	ed.Start = ed.Dot
	ed.End = ed.Dot
	ed.addr = -1
}
