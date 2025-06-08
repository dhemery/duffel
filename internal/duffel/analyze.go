package duffel

import (
	"encoding/json"
	"io/fs"
	"path"
	"path/filepath"
)

type Plan struct {
	Target string `json:"target"`
	Tasks  []Task `json:"tasks"`
}

func (p *Plan) Execute(fsys FS) error {
	for _, task := range p.Tasks {
		if err := task.Execute(fsys, p.Target); err != nil {
			return err
		}
	}
	return nil
}

type Task struct {
	// Item is the path of the item to create, relative to target.
	Item string
	// State describes the file to create at the target item path.
	State
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
}

// MarshalJSON implements [json.Marshaller].
// We have to implement it in order to override the [State.MarshalJSON] promoted from the embedded State,
// which marshals only the State fields.
func (t Task) MarshalJSON() ([]byte, error) {
	stateJSON, err := t.State.MarshalJSON()
	if err != nil {
		return nil, err
	}
	taskJSON, err := json.Marshal(struct {
		Item string `json:"item"`
	}{
		Item: t.Item,
	})
	if err != nil {
		return nil, err
	}
	// Replace the closing brace with a comma to continue with the state
	taskJSON[len(taskJSON)-1] = ','
	// Skip the the state's opening brace to continue after the task fields
	stateJSON = stateJSON[1:]
	return append(taskJSON, stateJSON...), nil
}

type ItemAnalyst interface {
	Analyze(pkg, item string, d fs.DirEntry) error
}

type PkgAnalyst struct {
	FS          fs.FS
	Pkg         string
	SourcePkg   string
	ItemAnalyst ItemAnalyst
}

func NewPkgAnalyst(fsys fs.FS, source, pkg string, a ItemAnalyst) PkgAnalyst {
	return PkgAnalyst{
		FS:          fsys,
		Pkg:         pkg,
		SourcePkg:   path.Join(source, pkg),
		ItemAnalyst: a,
	}
}

func (pa PkgAnalyst) Analyze() error {
	return fs.WalkDir(pa.FS, pa.SourcePkg,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Don't visit SourcePkg. It's a pkg, not an item.
			if path == pa.SourcePkg {
				return nil
			}
			item, _ := filepath.Rel(pa.SourcePkg, path)
			return pa.ItemAnalyst.Analyze(pa.Pkg, item, d)
		})
}
