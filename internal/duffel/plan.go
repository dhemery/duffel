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

type ItemVisitor interface {
	VisitItem(pkg, item string, d fs.DirEntry) error
}

type PkgPlanner struct {
	FS        fs.FS
	Source    string
	Pkg       string
	SourcePkg string
	Visitor   ItemVisitor
}

func NewPkgPlanner(fsys fs.FS, source, pkg string, v ItemVisitor) PkgPlanner {
	return PkgPlanner{
		FS:        fsys,
		Source:    source,
		Pkg:       pkg,
		SourcePkg: path.Join(source, pkg),
		Visitor:   v,
	}
}

func (p PkgPlanner) Plan() error {
	return fs.WalkDir(p.FS, p.SourcePkg, p.VisitPath)
}

func (p PkgPlanner) VisitPath(pth string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	// Don't visit SourcePkg. It's a pkg, not an item.
	if pth == p.SourcePkg {
		return nil
	}
	item, _ := filepath.Rel(p.SourcePkg, pth)
	return p.Visitor.VisitItem(p.Pkg, item, d)
}
