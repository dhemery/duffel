package analyze

import (
	"errors"
)

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

func sameNullity(l, r any) bool {
	lNil := l == nil
	rNil := r == nil
	return lNil == rNil
}
