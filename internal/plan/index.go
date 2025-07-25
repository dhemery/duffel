package plan

import (
	"fmt"
	"iter"
	"log/slog"
	"maps"

	"github.com/dhemery/duffel/internal/log"
)

type Stater interface {
	State(name string) (*State, error)
}

// Spec describes the current and planned states
// of a file in the target tree.
type Spec struct {
	Current *State
	Planned *State
}

// NewIndex returns a new, empty index that retrieves current file
// states by calling files.State.
func NewIndex(files Stater) *index {
	return &index{
		specs: map[string]Spec{},
		files: files,
	}
}

type index struct {
	specs map[string]Spec
	files Stater
}

// State returns the planned state of the named file.
// If i does not already know the planned state,
// State retrieves the current state of the file,
// stores it as both the current and planned states,
// and returns the retrieved state.
func (i *index) State(name string) (*State, error) {
	spec, ok := i.specs[name]
	if !ok {
		state, err := i.files.State(name)
		if err != nil {
			return nil, err
		}
		log.Info("record file state",
			slog.String("name", name),
			slog.Any("state", state),
		)
		spec = Spec{state, state}
		i.specs[name] = spec
	}
	return spec.Planned, nil
}

// SetState sets the planned state of the named file.
func (i *index) SetState(name string, state *State) {
	spec, ok := i.specs[name]
	if !ok {
		panic(fmt.Errorf("index.SetState(%q,_): no such spec", name))
	}
	log.Info("set planned state",
		slog.String("name", name),
		slog.Any("state", state),
	)

	spec.Planned = state
	i.specs[name] = spec
}

// All returns an iterator over the indexed specs.
func (i *index) All() iter.Seq2[string, Spec] {
	return maps.All(i.specs)
}
