package plan

import (
	"iter"
	"maps"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

// Stater describes the states of files.
type Stater interface {
	// State returns the state of the named file.
	State(name string) (*file.State, error)
}

// NewIndex returns a new, empty index that retrieves current file
// states by calling files.State.
func NewIndex(files Stater) *index {
	return &index{
		states: map[string]*file.State{},
		files:  files,
	}
}

type index struct {
	states map[string]*file.State
	files  Stater
}

// State returns the planned state of named file.
// If i does not already know the planned state,
// State retrieves the current state of the file,
// stores it as both the current and planned states,
// and returns the retrieved state.
func (i *index) State(name string) (*file.State, error) {
	var err error
	state, ok := i.states[name]
	if !ok {
		state, err = i.files.State(name)
		if err != nil {
			return nil, err
		}
		i.states[name] = state
	}
	return state, nil
}

// SetState sets the planned state of named file.
func (i *index) SetState(name string, state *file.State) {
	i.states[name] = state
}

// All returns an iterator over the states in name order.
func (i *index) All() iter.Seq2[string, *file.State] {
	return func(yield func(string, *file.State) bool) {
		for _, name := range slices.Sorted(maps.Keys(i.states)) {
			if !yield(name, i.states[name]) {
				return
			}
		}
	}
}
