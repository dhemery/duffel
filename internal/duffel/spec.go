package duffel

import (
	"fmt"

	"github.com/dhemery/duffel/internal/file"
)

// An Index collects Specs by item name.
type Index map[string]Spec

// A Spec describes the current and desired states of a target item file.
type Spec struct {
	Current *file.State `json:"current,omitzero"`
	Desired *file.State `json:"desired,omitzero"`
}

func (s Spec) String() string {
	return fmt.Sprintf("%T{Current:%v,Desired:%v}", s, s.Current, s.Desired)
}
