package analyze

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

type Merger interface {
	Merge(string, *slog.Logger) error
}

func NewInstall(target string, merger Merger) *Install {
	return &Install{
		target: target,
		merger: merger,
	}
}

// Install describes the installed states
// of the target files that correspond to the given package items.
type Install struct {
	target string
	merger Merger
}

// Apply returns the state of the targetItem file
// that would result from installing the sourceItem file.
func (op Install) Apply(s SourceItem, t TargetItem, l *slog.Logger) (*file.State, error) {
	itemAsDest := t.Path.PathTo(s.Path.String())
	if t.State == nil {
		// There is no target item, so we're free to create a link to the pkg item.
		var err error
		if s.Entry.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return &file.State{
			Type:     fs.ModeSymlink,
			Dest:     itemAsDest,
			DestType: s.Entry.Type(),
		}, err
	}

	// At this point, we know that the target exists,
	// either on the file system or as planned by a previous operation.

	if t.State.Type.IsRegular() {
		// Cannot modify an existing regular file.
		return nil, conflictError(s, t)
	}

	if t.State.Type.IsDir() {
		if s.Entry.IsDir() {
			// The target and pkg item are both dirs.
			// Return the target state unchanged,
			// and a nil error to walk the pkg item's contents.
			return t.State, nil
		}

		// The target is a dir, but the pkg item is not.
		// Cannot merge the target dir with a non-dir pkg item.
		return nil, conflictError(s, t)
	}

	if t.State.Type.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, conflictError(s, t)
	}

	// At this point, we know that the existing target is a symlink.

	if t.State.Dest == itemAsDest {
		// The target already links to this pkg item.
		// There's nothing more to do.
		var err error
		if s.Entry.IsDir() {
			// We're done with this item. Do not walk its contents.
			err = fs.SkipDir
		}
		return t.State, err
	}

	if !t.State.DestType.IsDir() {
		// The target's link destination is not a dir. Cannot merge.
		return nil, conflictError(s, t)
	}

	if !s.Entry.IsDir() {
		// Tne entry is not a dir. Cannot merge.
		return nil, conflictError(s, t)
	}

	// The package item is a dir and the target is a link to a dir.
	// Try to merge the target item.
	err := op.merger.Merge(t.Path.Resolve(t.State.Dest), l)
	if err != nil {
		return nil, err
	}

	// No conflicts installing the target destination's contents.
	// Now change the target to a dir, and walk the current item
	// to install its contents into the dir.
	dirState := &file.State{Type: fs.ModeDir}
	return dirState, nil
}

func conflictError(s SourceItem, t TargetItem) error {
	return &ConflictError{
		Item:        s.Path,
		ItemType:    s.Entry.Type(),
		Target:      t.Path,
		TargetState: t.State,
	}
}

type ConflictError struct {
	Item        SourcePath
	ItemType    fs.FileMode
	Target      TargetPath
	TargetState *file.State
}

func (ce *ConflictError) Error() string {
	return fmt.Sprintf("install conflict: package item %q is %s, target %q is %s",
		ce.Item, typeString(ce.ItemType), ce.Target, stateString(ce.TargetState))
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
