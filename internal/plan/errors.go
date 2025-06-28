package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

var (
	ErrDestNotPkgItem = errors.New("destination is not a package item")
	ErrIsDir          = errors.New("is a directory")
)

type TargetError struct {
	Op    string
	Pkg   string
	Item  string
	State *file.State
}

func (e *TargetError) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s %q: cannot alter target %q: target is %s",
		e.Op, pkgItem, e.Item, stateString(e.State))
}

type ConflictError struct {
	Op          string
	Pkg         string
	Item        string
	ItemType    fs.FileMode
	TargetState *file.State
}

func (e *ConflictError) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s %q: cannot replace or merge target %q with package item %q: target is %s, package item is %s",
		e.Op, pkgItem, e.Item, pkgItem, stateString(e.TargetState), modeTypeString(e.ItemType))
}

func modeTypeString(t fs.FileMode) string {
	switch {
	case t.IsRegular():
		return "a regular file"
	case t.IsDir():
		return "a directory"
	case t&fs.ModeSymlink != 0:
		return "a symlink"
	default:
		return fmt.Sprintf("unknown file type %s", t.String())
	}
}

func stateString(s *file.State) string {
	if s.Mode&fs.ModeSymlink != 0 {
		return fmt.Sprintf("%s to %s (%s)", modeTypeString(s.Mode), modeTypeString(s.DestMode), s.Dest)
	}
	return modeTypeString(s.Mode)
}
