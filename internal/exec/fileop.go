package exec

import (
	"fmt"
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
)

var (
	OpMkdir   = "mkdir"
	OpRemove  = "remove"
	OpSymlink = "symlink"
)

type FileOp struct {
	Op   string `json:"op"`
	Dest string `json:"dest,omitempty"`
}

func (op FileOp) Execute(fsys fs.FS, name string) error {
	switch op.Op {
	case OpMkdir:
		return file.Mkdir(fsys, name, fs.ModeDir|0o755)
	case OpRemove:
		return file.Remove(fsys, name)
	case OpSymlink:
		return file.Symlink(fsys, op.Dest, name)
	}
	return fmt.Errorf("unknown file op %q", op.Op)
}
