package plan

import (
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type ItemOp interface {
	Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error)
}

type StateIndex interface {
	Desired(item string) (*file.State, error)
	SetDesired(item string, state *file.State)
}

type PkgWalker struct {
	FS      fs.FS
	WalkDir string
	Pkg     string
	Index   StateIndex
	ItemOp  ItemOp
}

func NewPkgWalker(fsys fs.FS, target, source, pkg string, index StateIndex, itemOp ItemOp) PkgWalker {
	return PkgWalker{
		FS:      fsys,
		WalkDir: path.Join(source, pkg),
		Pkg:     pkg,
		Index:   index,
		ItemOp:  itemOp,
	}
}

func (pa PkgWalker) Walk() error {
	return fs.WalkDir(pa.FS, pa.WalkDir, pa.VisitPath)
}

func (pa PkgWalker) VisitPath(name string, entry fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if name == pa.WalkDir {
		// Skip the dir being walked. It is not an item.
		return nil
	}

	item := name[len(pa.WalkDir)+1:]
	priorState, err := pa.Index.Desired(item)
	if err != nil {
		return err
	}

	newState, err := pa.ItemOp.Apply(pa.Pkg, item, entry, priorState)
	if err != nil {
		return err
	}

	pa.Index.SetDesired(item, newState)
	return nil
}
