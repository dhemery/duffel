package item

import (
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

type MissFunc func(item string) (*file.State, error)

// An Index collects Specs by item name.
type Index struct {
	specs map[string]Spec
	miss  MissFunc
}

// NewIndex returns a new, empty index that retrieves missing items via miss.
func NewIndex(miss MissFunc) *Index {
	return &Index{
		specs: map[string]Spec{},
		miss:  miss,
	}
}

// Desired returns the desired state of item.
func (i *Index) Desired(item string) (*file.State, error) {
	spec, ok := i.specs[item]
	if !ok {
		state, err := i.miss(item)
		if err != nil {
			return nil, err
		}
		spec = Spec{Current: state, Desired: state}
		i.specs[item] = spec
	}
	return spec.Desired, nil
}

// SetDesired sets the desired state of item to state.
func (i *Index) SetDesired(item string, state *file.State) {
	spec := i.specs[item]
	spec.Desired = state
	i.specs[item] = spec
}

// ByItem returns an iterator over the item/spec pairs in i.
func (i *Index) ByItem() iter.Seq2[string, Spec] {
	return func(yield func(string, Spec) bool) {
		for _, item := range slices.Sorted(maps.Keys(i.specs)) {
			if !yield(item, i.specs[item]) {
				return
			}
		}
	}
}

// A Spec describes the current and desired states of an item.
type Spec struct {
	Current *file.State `json:"current,omitzero"`
	Desired *file.State `json:"desired,omitzero"`
}

func (s Spec) String() string {
	return fmt.Sprintf("%T{Current:%v,Desired:%v}", s, s.Current, s.Desired)
}
