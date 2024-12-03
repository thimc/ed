package main

import (
	"strings"
)

type cursor struct {
	first  int
	second int
	dot    int // current address
	addrc  int // address count
}

type file struct {
	dirty bool           // modified state
	lines []string       // file content
	mark  ['z' - 'a']int // a to z
	path  string         // full file path to the file
}

func (f *file) append(dest int, lines []string) {
	f.lines = append(f.lines[:dest], append(lines, f.lines[dest:]...)...)
}

func (f *file) yank(start, end, dest int) int {
	buf := make([]string, end-start+1)
	copy(buf, f.lines[start-1:end])
	f.lines = append(f.lines[:dest], append(buf, f.lines[dest:]...)...)
	return len(buf)
}

func (f *file) delete(start, end int) {
	f.lines = append(f.lines[:start-1], f.lines[end:]...)
}

func (f *file) join(start, end int) {
	buf := strings.Join(f.lines[start-1:end], "")
	f.lines = append(append(append([]string{}, f.lines[:start-1]...), buf), f.lines[end:]...)
}

func (f *file) move(start, end, dest int) int {
	buf := make([]string, end-start+1)
	copy(buf, f.lines[start-1:end])
	f.lines = append(f.lines[:start-1], f.lines[end:]...) // remove the lines
	if dest > start {
		dest -= (end - start + 1)
	}
	f.lines = append(f.lines[:dest], append(buf, f.lines[dest:]...)...)
	return dest + (end - start + 1)
}
