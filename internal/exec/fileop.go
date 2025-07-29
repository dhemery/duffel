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

func (op FileOp) Execute(fsys fs.FS, target string) error {
	switch op.Op {
	case OpMkdir:
		return file.Mkdir(fsys, target, fs.ModeDir|0o755)
	case OpRemove:
		return file.Remove(fsys, target)
	case OpSymlink:
		return file.Symlink(fsys, op.Dest, target)
	}
	return fmt.Errorf("unknown file op %q", op.Op)
}
