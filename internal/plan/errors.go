package plan

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
		e.Item, modeTypeString(e.ItemType), e.Target, stateString(e.TargetState))
}

func modeTypeString(m fs.FileMode) string {
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
	if s.Mode&fs.ModeSymlink != 0 {
		return fmt.Sprintf("%s to %s (%s)", modeTypeString(s.Mode), modeTypeString(s.DestMode), s.Dest)
	}
	return modeTypeString(s.Mode)
}
