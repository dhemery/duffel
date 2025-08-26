package plan

import (
	"iter"
	"log/slog"
	"maps"

	"github.com/dhemery/duffel/internal/file"
)

// A spec describes the current and planned states of a file in the target tree.
type spec struct {
	current file.State
	planned file.State
}

// newIndex returns a new, empty specIndex that reads file states from s.
func newIndex(s stater) *specIndex {
	return &specIndex{
		specs:  map[string]spec{},
		stater: s,
	}
}

// A specIndex maintains a spec for each known file.
type specIndex struct {
	specs  map[string]spec
	stater stater
}

type stater interface {
	// State returns the state of the named file.
	State(name string) (file.State, error)
}

// state returns the planned state of the named file.
// If i does not already know the planned state,
// this method reads the current state of the file,
// stores it as both the current and planned states,
// and returns the state.
func (i *specIndex) state(t TargetPath, l *slog.Logger) (file.State, error) {
	name := t.String()
	s, ok := i.specs[name]
	if !ok {
		state, err := i.stater.State(name)
		if err != nil {
			return file.State{}, err
		}
		attrs := slog.GroupAttrs("target", slog.Any("path", t), slog.Any("file_state", state))
		l.Debug("read target file state", attrs)
		s = spec{state, state}
		i.specs[name] = s
	}
	return s.planned, nil
}

// setState sets the planned state of the target file.
func (i *specIndex) setState(t TargetPath, s file.State, l *slog.Logger) {
	name := t.String()
	spec := i.specs[name]
	attrs := slog.GroupAttrs("target", slog.Any("path", t), slog.Any("planned_state", s))
	l.Info("set target planned state", attrs)
	spec.planned = s
	i.specs[name] = spec
}

// all returns an iterator over the indexed specs.
func (i *specIndex) all() iter.Seq2[string, spec] {
	return maps.All(i.specs)
}
