package analyze

import (
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/dhemery/duffel/internal/file"
)

type Merger interface {
	Merge(string) error
}

func NewInstall(target string, merger Merger, logger *slog.Logger) *Install {
	return &Install{
		target: target,
		merger: merger,
		logger: logger,
	}
}

// Install describes the installed states
// of the target files that correspond to the given package items.
type Install struct {
	target string
	merger Merger
	logger *slog.Logger
}

// Apply returns the state of the targetItem file
// that would result from installing the sourceItem file.
// Entry describes the state of the item file in the source tree.
// TargetState describes the state of targetItem as planned by earlier analysis.
func (op Install) Apply(sourceItem SourcePath, entry fs.DirEntry, targetItem TargetPath, targetState *file.State) (*file.State, error) {
	sg := slog.Group("source", "path", sourceItem, "entry", entry)
	tg := slog.Group("target", "path", targetItem, "state", targetState)
	op.logger.Info("install", sg, tg)
	itemAsDest := targetItem.PathTo(sourceItem.String())
	if targetState == nil {
		// There is no target item, so we're free to create a link to the pkg item.
		var err error
		if entry.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return &file.State{
			Type:     fs.ModeSymlink,
			Dest:     itemAsDest,
			DestType: entry.Type(),
		}, err
	}

	// At this point, we know that the target exists,
	// either on the file system or as planned by a previous operation.

	if targetState.Type.IsRegular() {
		// Cannot modify an existing regular file.
		return nil, conflictError(sourceItem, entry, targetItem, targetState)
	}

	if targetState.Type.IsDir() {
		if entry.IsDir() {
			// The target and pkg item are both dirs.
			// Return the target state unchanged,
			// and a nil error to walk the pkg item's contents.
			return targetState, nil
		}

		// The target is a dir, but the pkg item is not.
		// Cannot merge the target dir with a non-dir pkg item.
		return nil, conflictError(sourceItem, entry, targetItem, targetState)
	}

	if targetState.Type.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, conflictError(sourceItem, entry, targetItem, targetState)
	}

	// At this point, we know that the existing target is a symlink.

	if targetState.Dest == itemAsDest {
		// The target already links to this pkg item.
		// There's nothing more to do.
		var err error
		if entry.IsDir() {
			// We're done with this item. Do not walk its contents.
			err = fs.SkipDir
		}
		return targetState, err
	}

	if !targetState.DestType.IsDir() {
		// The target's link destination is not a dir. Cannot merge.
		return nil, conflictError(sourceItem, entry, targetItem, targetState)
	}

	if !entry.IsDir() {
		// Tne entry is not a dir. Cannot merge.
		return nil, conflictError(sourceItem, entry, targetItem, targetState)
	}

	// The package item is a dir and the target is a link to a dir.
	// Try to merge the target item.
	err := op.merger.Merge(targetItem.Resolve(targetState.Dest))
	if err != nil {
		return nil, err
	}

	// No conflicts installing the target destination's contents.
	// Now change the target to a dir, and walk the current item
	// to install its contents into the dir.
	dirState := &file.State{Type: fs.ModeDir}
	return dirState, nil
}

func conflictError(item SourcePath, entry fs.DirEntry, target TargetPath, state *file.State) error {
	return &ConflictError{
		Item:        item,
		ItemType:    entry.Type(),
		Target:      target,
		TargetState: state,
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
