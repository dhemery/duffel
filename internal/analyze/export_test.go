package analyze

import (
	"errors"
)

func (s SourcePath) Equal(o SourcePath) bool {
	return s.s == o.s &&
		s.p == o.p &&
		s.i == o.i
}

func (t TargetPath) Equal(o TargetPath) bool {
	return t.t == o.t &&
		t.i == o.i
}

func (me *MergeError) Equal(o *MergeError) bool {
	if !sameNullity(me, o) {
		return false
	}
	if me == nil {
		return true
	}
	return me.Name == o.Name &&
		errors.Is(me.Err, o.Err)
}

func (ce *ConflictError) Equal(o *ConflictError) bool {
	if !sameNullity(ce, o) {
		return false
	}
	if ce == nil {
		return true
	}
	return ce.ItemType == o.ItemType &&
		ce.Item.Equal(o.Item) &&
		ce.Target.Equal(o.Target) &&
		ce.TargetState.Equal(o.TargetState)
}

func sameNullity(l, r any) bool {
	lNil := l == nil
	rNil := r == nil
	return lNil == rNil
}
