package plan

import (
	"io/fs"
	"path"
)

type Index interface {
	State(name string) (*State, error)
	SetState(item string, state *State)
}

type PkgOp interface {
	WalkDir() string
	VisitFunc(target string, index Index) fs.WalkDirFunc
}

func NewAnalyst(fsys fs.FS, target string, index *index) analyst {
	return analyst{fsys: fsys, target: target, index: index}
}

type analyst struct {
	fsys   fs.FS
	target string
	index  *index
}

func (a analyst) Analyze(ops ...PkgOp) (*index, error) {
	for _, op := range ops {
		err := fs.WalkDir(a.fsys, op.WalkDir(), op.VisitFunc(a.target, a.index))
		if err != nil {
			return nil, err
		}
	}
	return a.index, nil
}

type ItemOp interface {
	Apply(name string, entry fs.DirEntry, indexState *State) (*State, error)
}

func NewPkgOp(pkgDir string, itemOp ItemOp) pkgOp {
	return pkgOp{
		walkDir: pkgDir,
		pkgDir:  pkgDir,
		itemOp:  itemOp,
	}
}

func NewMergePkgOp(pkgDir, mergeItem string, itemOp ItemOp) pkgOp {
	return pkgOp{
		pkgDir:  pkgDir,
		walkDir: path.Join(pkgDir, mergeItem),
		itemOp:  itemOp,
	}
}

type pkgOp struct {
	pkgDir  string
	walkDir string
	itemOp  ItemOp
}

func (po pkgOp) WalkDir() string {
	return po.walkDir
}

func (po pkgOp) VisitFunc(target string, index Index) fs.WalkDirFunc {
	return func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == po.walkDir {
			// Skip the dir being walked.
			return nil
		}

		item := name[len(po.pkgDir)+1:]
		targetItem := path.Join(target, item)
		oldState, err := index.State(targetItem)
		if err != nil {
			return err
		}

		newState, err := po.itemOp.Apply(name, entry, oldState)

		if err == nil || err == fs.SkipDir {
			index.SetState(targetItem, newState)
		}

		return err
	}
}
