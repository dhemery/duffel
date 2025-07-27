package analyze

import (
	"fmt"
	"io/fs"

	"github.com/dhemery/duffel/internal/file"
)

type InstallError struct {
	Item        string
	ItemType    fs.FileMode
	Target      string
	TargetState *file.State
}

func (e *InstallError) Error() string {
	return fmt.Sprintf("install conflict: package item %q is %s, target %q is %s",
		e.Item, typeString(e.ItemType), e.Target, stateString(e.TargetState))
}

func typeString(m fs.FileMode) string {
	switch {
	case m.IsRegular():
		return "a regular file"
	case m.IsDir():
		return "a directory"
	case m&fs.ModeSymlink != 0:
		return "a symlink"
	default:
		return fmt.Sprintf("unknown file type %s", m.String())
	}
}

func stateString(s *file.State) string {
	if s == nil {
		return "<nil>"
	}
	if s.Type&fs.ModeSymlink != 0 {
		return fmt.Sprintf("%s to %s (%s)", typeString(s.Type), typeString(s.DestType), s.Dest)
	}
	return typeString(s.Type)
}
