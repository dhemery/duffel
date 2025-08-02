package analyze

import (
	"iter"
	"log/slog"
	"maps"

	"github.com/dhemery/duffel/internal/file"
)

type Stater interface {
	State(name string) (file.State, error)
}

// Spec describes the current and planned states of a file in the target tree.
type Spec struct {
	Current file.State
	Planned file.State
}

// NewIndex returns a new, empty index that reads file system states from s.
func NewIndex(s Stater, l *slog.Logger) *index {
	return &index{
		specs:  map[string]Spec{},
		stater: s,
	}
}

type index struct {
	specs  map[string]Spec
	stater Stater
}

// State returns the planned state of the named file.
// If i does not already know the planned state,
// this method reads the current state of the file,
// stores it as both the current and planned states,
// and returns the state.
func (i *index) State(name string, logger *slog.Logger) (file.State, error) {
	spec, ok := i.specs[name]
	if !ok {
		state, err := i.stater.State(name)
		if err != nil {
			return state, err
		}
		logger.Info("read target file state", "file_state", state)
		spec = Spec{state, state}
		i.specs[name] = spec
	}
	return spec.Planned, nil
}

// SetState sets the planned state of the named file.
func (i *index) SetState(name string, state file.State, logger *slog.Logger) {
	spec := i.specs[name]
	logger.Info("updated goal state", "goal_state", state)
	spec.Planned = state
	i.specs[name] = spec
}

// All returns an iterator over the indexed specs.
func (i *index) All() iter.Seq2[string, Spec] {
	return maps.All(i.specs)
}
