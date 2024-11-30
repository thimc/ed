package main

import "log"

// undoType determines how the history entry should behandled,
// undoTypeDelete removes lines and undoTypeAdd adds lines.
type undoType int

const (
	undoTypeAdd undoType = iota
	undoTypeDelete
)

type undoAction struct {
	cursor
	typ   undoType
	lines []string
}

type undo struct {
	action  []undoAction // history for a command in progress
	history [][]undoAction
	global  []undoAction
}

func (u *undo) clear() { u.action = []undoAction{} }

func (u *undo) reset() {
	u.clear()
	u.history = [][]undoAction{}
}
func (u *undo) pop(ed *Editor) error {
	if len(u.history) < 1 {
		return ErrNothingToUndo
	}
	action := u.history[len(u.history)-1]
	u.history = u.history[:len(u.history)-1]
	for i := len(action) - 1; i >= 0; i-- {
		a := action[i]
		before := ed.file.lines[:a.first-1]
		after := ed.file.lines[a.first-1:]
		log.Printf("%+v", a)
		switch a.typ {
		case undoTypeDelete:
			after = ed.file.lines[a.second:]
			ed.file.lines = append(before, after...)
		case undoTypeAdd:
			ed.file.lines = append(before, append(a.lines, after...)...)
		}
		ed.dot = a.dot
		ed.file.dirty = true
	}
	if count := len(u.history) - 1; count > 0 {
		u.history = u.history[:count]
	} else {
		u.history = [][]undoAction{}
	}
	return nil
}

func (u *undo) append(typ undoType, start, end, dot int, lines []string) {
	u.action = append(u.action, undoAction{
		typ:    typ,
		cursor: cursor{first: start, second: end, dot: dot},
		lines:  lines,
	})
}

func (u *undo) store(g bool) {
	if g {
		u.global = append(u.global, u.action...)
	} else {
		u.history = append(u.history, u.action)
	}
	u.clear()
}

func (u *undo) storeGlobal() {
	u.history = append(u.history, u.global)
	u.clear()
}
