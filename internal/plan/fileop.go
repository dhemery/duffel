package plan

import (
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
)

func NewSymlinkOp(dest string) SymlinkOp {
	return SymlinkOp{Op: "symlink"}
}

type SymlinkOp struct {
	Op   string `json:"op"`
	Dest string `json:"dest,omitempty"`
}

func (op SymlinkOp) Execute(fsys fs.FS, target string) error {
	return file.Symlink(fsys, op.Dest, target)
}

var MkDirOp = mkDirOp{"mkdir"}

type mkDirOp struct {
	Op string
}

func (op mkDirOp) Execute(fsys fs.FS, target string) error {
	return file.MkDir(fsys, target, fs.ModeDir|0o755)
}
