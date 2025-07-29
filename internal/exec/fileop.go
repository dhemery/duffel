package exec

import (
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
)

func NewSymlinkOp(dest string) SymlinkOp {
	return SymlinkOp{Op: "symlink", Dest: dest}
}

type SymlinkOp struct {
	Op   string `json:"op"`
	Dest string `json:"dest"`
}

func (op SymlinkOp) Execute(fsys fs.FS, target string) error {
	return file.Symlink(fsys, op.Dest, target)
}

var MkDirOp = mkDirOp("mkdir")

type mkDirOp string

func (op mkDirOp) Execute(fsys fs.FS, target string) error {
	return file.Mkdir(fsys, target, fs.ModeDir|0o755)
}

var RemoveOp = removeOp("remove")

type removeOp string

func (op removeOp) Execute(fsys fs.FS, target string) error {
	return file.Remove(fsys, target)
}
