package item

import (
	"fmt"
	"io/fs"
	"iter"
	"maps"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

// An Index collects Specs by item name.
type Index struct {
	specs map[string]Spec
}

// NewIndex returns a new, empty index.
func NewIndex() *Index {
	return &Index{specs: map[string]Spec{}}
}

// Set associates spec with item in i,
// replacing any earlier association.
func (i *Index) Set(item string, spec Spec) {
	if i.specs == nil {
		i.specs = make(map[string]Spec)
	}
	i.specs[item] = spec
}

// Get returns the spec associated with item.
func (i *Index) Get(item string) (Spec, error) {
	s, ok := i.specs[item]
	if !ok {
		return s, fs.ErrNotExist
	}
	return s, nil
}

// ByItem returns an iterator over the item/spec pairs in i.
func (i *Index) ByItem() iter.Seq2[string, Spec] {
	return func(yield func(string, Spec) bool) {
		for _, name := range slices.Sorted(maps.Keys(i.specs)) {
			if !yield(name, i.specs[name]) {
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
