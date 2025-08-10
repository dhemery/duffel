package plan

import (
	"fmt"
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
)

var (
	ActMkdir   = "mkdir"
	ActRemove  = "remove"
	ActSymlink = "symlink"
)

type Action struct {
	Action string `json:"action"`
	Dest   string `json:"dest,omitempty"`
}

func (a Action) Execute(fsys fs.FS, name string) error {
	switch a.Action {
	case ActMkdir:
		return file.Mkdir(fsys, name, fs.ModeDir|0o755)
	case ActRemove:
		return file.Remove(fsys, name)
	case ActSymlink:
		return file.Symlink(fsys, a.Dest, name)
	}
	return fmt.Errorf("unknown file action %q", a.Action)
}
