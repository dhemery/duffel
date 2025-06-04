package duffel

import (
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
	// Item is the path of the item to create, relative to target
	Item string `json:"item"`

	// State describes the file to create at the target item path
	State
}

func (t Task) Execute(fsys FS, target string) error {
	return fsys.Symlink(t.Dest, path.Join(target, t.Item))
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

func (pa PkgAnalyst) Plan() error {
	return fs.WalkDir(pa.FS, pa.SourcePkg, pa.Analyze)
}

func (pa PkgAnalyst) Analyze(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	// Don't visit SourcePkg. It's a pkg, not an item.
	if path == pa.SourcePkg {
		return nil
	}
	item, _ := filepath.Rel(pa.SourcePkg, path)
	return pa.ItemAnalyst.Analyze(pa.Pkg, item, d)
}
