package main

type undoType int

const (
	undoAdd undoType = iota
	undoDelete
)

type undoAction struct {
	cursor
	typ   undoType
	lines []string
}

type undo struct {
	action  []undoAction
	history [][]undoAction
}

func (u *undo) clear() {
	u.history = [][]undoAction{}
	u.action = []undoAction{}
}

func (u *undo) pop() error {
	if len(u.history) < 1 {
		return ErrNothingToUndo
	}
	// TODO: undo pop (apply)
	return nil
}

func (u *undo) push(typ undoType, start, end, dot int, lines []string) {
	u.action = append(u.action, undoAction{
		typ:    typ,
		cursor: cursor{first: start, second: end, dot: dot},
		lines:  lines,
	})
	// TODO: undo push (store)
}

func (u *undo) store() {
	u.history = append(u.history, u.action)
	u.action = []undoAction{}
}
