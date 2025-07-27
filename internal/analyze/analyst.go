package analyze

import (
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
)

type Index interface {
	State(name string) (*file.State, error)
	SetState(item string, state *file.State)
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
