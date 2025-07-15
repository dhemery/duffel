package plan

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

type InstallError struct {
	Op          string
	Pkg         string
	Item        string
	ItemType    fs.FileMode
	TargetState *file.State
}

func (e *InstallError) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s %q conflict: target %q is %s, package item %q is %s",
		e.Op, pkgItem, e.Item, stateString(e.TargetState), pkgItem, modeTypeString(e.ItemType))
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
