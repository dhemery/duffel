package plan

import (
	"fmt"
	"io/fs"
)

var (
	ActMkdir   = "mkdir"   // Create a directory with permission 0o755.
	ActRemove  = "remove"  // Remove a file or (empty) directory.
	ActSymlink = "symlink" // Create a symlink.
)

// ActionFS provides methods to execute actions in a file system.
type ActionFS interface {
	// Mkdir creates a new directory with the specified name and permission bits
	Mkdir(name string, perm fs.FileMode) error

	// Remove removes the named file or (empty) directory.
	Remove(name string) error

	// Symlink creates newname as a symbolic link to oldname.
	Symlink(oldname, newname string) error
}

// Action describes a change to make to a file.
type Action struct {
	// Action is the kind of change to make.
	Action string `json:"action"`

	// Dest is the link destination if the action is [ActSymlink].
	Dest string `json:"dest,omitempty"`
}

// Execute performs the action on the named file.
func (a Action) Execute(afs ActionFS, name string) error {
	switch a.Action {
	case ActMkdir:
		return afs.Mkdir(name, fs.ModeDir|0o755)
	case ActRemove:
		return afs.Remove(name)
	case ActSymlink:
		return afs.Symlink(a.Dest, name)
	}
	return fmt.Errorf("unknown file action %q", a.Action)
}
