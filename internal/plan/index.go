package plan

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
func NewIndex(s Stater) *index {
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
func (i *index) State(t TargetPath, l *slog.Logger) (file.State, error) {
	name := t.String()
	spec, ok := i.specs[name]
	if !ok {
		state, err := i.stater.State(name)
		if err != nil {
			return file.State{}, err
		}
		attrs := slog.GroupAttrs("target", slog.Any("path", t), slog.Any("file_state", state))
		l.Debug("read target file state", attrs)
		spec = Spec{state, state}
		i.specs[name] = spec
	}
	return spec.Planned, nil
}

// SetState sets the planned state of the target file.
func (i *index) SetState(t TargetPath, s file.State, l *slog.Logger) {
	name := t.String()
	spec := i.specs[name]
	attrs := slog.GroupAttrs("target", slog.Any("path", t), slog.Any("planned_state", s))
	l.Info("set target planned state", attrs)
	spec.Planned = s
	i.specs[name] = spec
}

// All returns an iterator over the indexed specs.
func (i *index) All() iter.Seq2[string, Spec] {
	return maps.All(i.specs)
}
