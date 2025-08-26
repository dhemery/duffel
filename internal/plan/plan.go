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

// NewPlanner returns a new [Planner]
// that plans how to achieive goals in the file tree rooted at target.
func NewPlanner(fsys fs.ReadLinkFS, target string, goals []DirGoal, l *slog.Logger) *Planner {
	stater := file.NewStater(fsys)
	index := newIndex(stater)
	analyst := newAnalyzer(fsys, target, index)
	return &Planner{target, analyst, goals, l}
}

// Execute returns a function that executes its [Plan] argument in the specified file system.
func Execute(fsys file.ActionFS, l *slog.Logger) func(p Plan) error {
	return func(p Plan) error {
		return p.execute(fsys, l)
	}
}

// Print returns a function that writes its [Plan] argument to w.
func Print(w io.Writer) func(p Plan) error {
	return func(p Plan) error {
		return p.print(w)
	}
}

// A Planner plans how to realize a set of goals in a target tree.
type Planner struct {
	target   string
	analyzer *analyzer
	goals    []DirGoal
	logger   *slog.Logger
}

// Plan creates a plan to realize p's goals in its target tree.
func (p Planner) Plan() (Plan, error) {
	for _, goal := range p.goals {
		if err := p.analyzer.analyze(goal, p.logger); err != nil {
			return Plan{}, err
		}
	}
	return newPlan(p.target, p.analyzer.index), nil
}

// A Plan is a sequence of tasks
// to bring the file tree rooted at target to the desired state.
type Plan struct {
	Target string          `json:"target"` // The root of the target file tree for the tasks to change.
	Tasks  map[string]Task `json:"tasks"`  // The file tasks to apply to the target.
}

// print writes the JSON encoding of the Plan to [io.Writer] w.
func (p Plan) print(w io.Writer) error {
	return json.MarshalWrite(w, &p, json.Deterministic(true))
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

// newPlan returns a [Plan] to bring the target tree
// to its planned state.
// Specs describes the current and planned state of each file.
func newPlan(target string, specs specs) Plan {
	targetLen := len(target) + 1
	p := Plan{Target: target, Tasks: map[string]Task{}}
	for name, spec := range specs.all() {
		if spec.current == spec.planned {
			continue
		}
		item := name[targetLen:]
		p.Tasks[item] = newTask(spec.current, spec.planned)
	}
	return p
}

// newTask creates a [Task] with the actions to bring file
// from the current state to the planned state.
func newTask(current, planned file.State) Task {
	t := Task{}

	switch {
	case current.IsNoFile(): // No-op
	case current.IsLink():
		t = append(t, file.RemoveAction())
	default:
		panic("do not know an action to remove " + current.String())
	}

	switch {
	case planned.IsNoFile(): // No-op
	case planned.IsDir():
		t = append(t, file.MkdirAction())
	case planned.IsLink():
		t = append(t, file.SymlinkAction(planned.Dest.Path))
	default:
		panic("do not know an action to create " + planned.String())
	}

	return t
}

// A Task is a sequence of actions to bring a file to a desired state.
type Task []file.Action

// Execute executes t's actions on the named file.
func (t Task) Execute(fsys file.ActionFS, name string) error {
	for _, a := range t {
		if err := a.Execute(fsys, name); err != nil {
			return err
		}
	}
	return nil
}

// specs is a collection that maps a file name to the spec for the file.
type specs interface {
	// all returns an iterator over the item name and [analyze.Spec]
	// for each file in the target tree that is not in its planned state.
	all() iter.Seq2[string, spec]
}
