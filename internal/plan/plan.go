package plan

import (
	"encoding/json/v2"
	"io"
	"io/fs"
	"iter"
	"log/slog"
	"maps"
	"path"
	"slices"

	"github.com/dhemery/duffel/internal/file"
)

// NewPlanner returns a planner that plans how to change the tree rooted at target.
func NewPlanner(fsys fs.ReadLinkFS, target string, goals []DirGoal, l *slog.Logger) *Planner {
	stater := file.NewStater(fsys)
	index := NewIndex(stater)
	analyst := NewAnalyzer(fsys, target, index)
	return &Planner{target, analyst, goals, l}
}

type Planner struct {
	target   string
	analyzer *analyzer
	goals    []DirGoal
	logger   *slog.Logger
}

// Plan creates a plan to realize ops in p's target tree.
func (p Planner) Plan() (Plan, error) {
	for _, goal := range p.goals {
		if err := p.analyzer.Analyze(goal, p.logger); err != nil {
			return Plan{}, err
		}
	}
	return NewPlan(p.target, p.analyzer.index), nil
}

// A Plan is a sequence of tasks
// to bring the file tree rooted at Target to the desired state.
type Plan struct {
	Target string          `json:"target"`
	Tasks  map[string]Task `json:"tasks"`
}

// Execute returns a function that executes its Plan argument in the specified file system.
func Execute(fsys file.ActionFS, l *slog.Logger) func(p Plan) error {
	return func(p Plan) error {
		return p.execute(fsys, l)
	}
}

// Print returns a function that writes its Plan argument to the specified [io.Writer].
func Print(w io.Writer) func(p Plan) error {
	return func(p Plan) error {
		return p.print(w)
	}
}

// print writes the JSON encoding of the Plan to [io.Writer] w.
func (p Plan) print(w io.Writer) error {
	return json.MarshalWrite(w, p, json.Deterministic(true))
}

// execute executes the Plan in [file.ActionFS] fsys.
func (p Plan) execute(fsys file.ActionFS, _ *slog.Logger) error {
	for _, item := range slices.Sorted(maps.Keys(p.Tasks)) {
		task := p.Tasks[item]
		name := path.Join(p.Target, item)
		if err := task.Execute(fsys, name); err != nil {
			return err
		}
	}
	return nil
}

// NewPlan returns a [Plan] to bring the target tree
// to its planned state.
// Specs describes the current and planned state of each file.
func NewPlan(target string, specs specs) Plan {
	targetLen := len(target) + 1
	p := Plan{Target: target, Tasks: map[string]Task{}}
	for name, spec := range specs.All() {
		if spec.Current == spec.Planned {
			continue
		}
		item := name[targetLen:]
		p.Tasks[item] = NewTask(spec.Current, spec.Planned)
	}
	return p
}

// NewTask creates a [Task] with the actions to bring file
// from the current state to the planned state.
func NewTask(current, planned file.State) Task {
	t := Task{}

	switch {
	case current.Type.IsNoFile(): // No-op
	case current.Type.IsLink():
		t = append(t, file.RemoveAction())
	default:
		panic("do not know an action to remove " + current.Type.String())
	}

	switch {
	case planned.Type.IsNoFile(): // No-op
	case planned.Type.IsDir():
		t = append(t, file.MkdirAction())
	case planned.Type.IsLink():
		t = append(t, file.SymlinkAction(planned.Dest.Path))
	default:
		panic("do not know an action to create " + planned.Type.String())
	}

	return t
}

// A Task is a sequence of actions to bring a file to a desired state.
type Task []file.Action

// Execute executes t's actions on the named file.
func (t Task) Execute(afs file.ActionFS, name string) error {
	for _, op := range t {
		if err := op.Execute(afs, name); err != nil {
			return err
		}
	}
	return nil
}

// specs is a collection that maps a file name to the Spec for the file.
type specs interface {
	// All returns an iterator over the item name and [analyze.Spec]
	// for each file in the target tree that is not in its planned state.
	All() iter.Seq2[string, Spec]
}
