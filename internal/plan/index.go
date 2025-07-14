package plan

import (
	"iter"
	"maps"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

type Stater interface {
	State(name string) (*file.State, error)
}

// NewIndex returns a new, empty index that retrieves missing items via miss.
func NewIndex(miss Stater) *index {
	return &index{
		states: map[string]*file.State{},
		miss:   miss,
	}
}

type index struct {
	states map[string]*file.State
	miss   Stater
}

// State returns the state of item in the index.
// If i does not already have a state for the item,
// Get stores and returns the state returned by miss.
func (i *index) State(item string) (*file.State, error) {
	var err error
	state, ok := i.states[item]
	if !ok {
		state, err = i.miss.State(item)
		if err != nil {
			return nil, err
		}
		i.states[item] = state
	}
	return state, nil
}

// SetState sets the cached state of item to state.
func (i *index) SetState(item string, state *file.State) {
	i.states[item] = state
}

// All returns an iterator over the states in item order.
func (i *index) All() iter.Seq2[string, *file.State] {
	return func(yield func(string, *file.State) bool) {
		for _, k := range slices.Sorted(maps.Keys(i.states)) {
			if !yield(k, i.states[k]) {
				return
			}
		}
	}
}
