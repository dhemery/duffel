package plan

import (
	"fmt"
	"io/fs"
)

var (
	ActMkdir   = "mkdir"
	ActRemove  = "remove"
	ActSymlink = "symlink"
)

type PlanFS interface {
	Mkdir(name string, mode fs.FileMode) error
	Remove(name string) error
	Symlink(oldname, newname string) error
}

type Action struct {
	Action string `json:"action"`
	Dest   string `json:"dest,omitempty"`
}

func (a Action) Execute(pfs PlanFS, name string) error {
	switch a.Action {
	case ActMkdir:
		return pfs.Mkdir(name, fs.ModeDir|0o755)
	case ActRemove:
		return pfs.Remove(name)
	case ActSymlink:
		return pfs.Symlink(a.Dest, name)
	}
	return fmt.Errorf("unknown file action %q", a.Action)
}
