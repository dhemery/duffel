package plan

import (
	"errors"
)

func (me *mergeError) Equal(o *mergeError) bool {
	if !sameNullity(me, o) {
		return false
	}
	if me == nil {
		return true
	}
	return me.Dir == o.Dir &&
		errors.Is(me.Err, o.Err)
}

func sameNullity(l, r any) bool {
	lNil := l == nil
	rNil := r == nil
	return lNil == rNil
}
