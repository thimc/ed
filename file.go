package main

import (
	"os"
	"slices"
	"strings"
)

type cursor struct {
	first  int
	second int
	dot    int // current address
	addrc  int // address count
}

type file struct {
	dirty  bool           // modified state
	binary bool           // TODO(thimc): binary mode; replace any NULL with newline and don't write a trailing \n
	lines  []string       // file content
	mark   ['z' - 'a']int // a to z
	path   string         // full file path to the file
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
	f.lines = append(append(slices.Clone(f.lines[:start-1]), buf), f.lines[end:]...)
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

func (f *file) write(path string, r rune, start, end int) (int, error) {
	perms := os.O_CREATE | os.O_RDWR | os.O_TRUNC
	if r == 'W' {
		perms = perms&^os.O_TRUNC | os.O_APPEND
	}
	file, err := os.OpenFile(path, perms, 0666)
	if err != nil {
		return -1, ErrCannotOpenFile
	}
	start = max(start-1, 0)
	lastline := ""
	if len(f.lines) > 0 && f.lines[end-1] != "" {
		lastline = "\n"
	}
	size, err := file.WriteString(strings.Join(f.lines[start:end], "\n") + lastline)
	if err != nil {
		return -1, ErrCannotWriteFile
	}
	if err := file.Close(); err != nil {
		return -1, ErrCannotCloseFile
	}
	return size, nil
}
