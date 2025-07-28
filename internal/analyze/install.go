package analyze

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

type Merger interface {
	Merge(name, target string) error
}

func NewInstallOp(source, target string, merger Merger, logger *slog.Logger) *installOp {
	return &installOp{
		target: target,
		source: source,
		merger: merger,
		log:    logger,
	}
}

// installOp is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type installOp struct {
	source string
	target string
	merger Merger
	log    *slog.Logger
}

// Apply describes the installed state
// of the target file that corresponds to the given package item.
// Name identifies the  packageitem to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the planned state of the target file
// as planned by earlier analysis.
func (op installOp) Apply(name string, entry fs.DirEntry, targetState *file.State) (*file.State, error) {
	pkgItem := name[len(op.source)+1:]
	pkg, item, _ := strings.Cut(pkgItem, "/")
	targetItem := path.Join(op.target, item)
	itemAsDest, _ := filepath.Rel(path.Dir(targetItem), name)
	sg := slog.Group("source",
		"root", op.source,
		"name", name,
		"pkg", pkg,
		"item", item,
		"entry", entry,
	)
	tg := slog.Group("target",
		"root", op.target,
		"item", targetItem,
		"old-state", targetState,
	)
	op.log.Info("install", "link-path", itemAsDest, sg, tg)

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
		return nil, conflictError(name, entry, targetItem, targetState)
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
		return nil, conflictError(name, entry, targetItem, targetState)
	}

	if targetState.Type.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, conflictError(name, entry, targetItem, targetState)
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
		return nil, conflictError(name, entry, targetItem, targetState)
	}

	if !entry.IsDir() {
		// Tne entry is not a dir. Cannot merge.
		return nil, conflictError(name, entry, targetItem, targetState)
	}

	// The package item is a dir and the target is a link to a dir.
	// Try to merge the target dest.
	fullDest := path.Join(path.Dir(targetItem), targetState.Dest)
	err := op.merger.Merge(fullDest, op.target)
	if err != nil {
		return nil, err
	}

	// No conflicts installing the target destination's contents.
	// Now change the target to a dir, and walk the current item
	// to install its contents into the dir.
	dirState := &file.State{Type: fs.ModeDir}
	return dirState, nil
}

func conflictError(item string, entry fs.DirEntry, target string, state *file.State) error {
	return &ConflictError{
		Item:        item,
		ItemType:    entry.Type(),
		Target:      target,
		TargetState: state,
	}
}

type ConflictError struct {
	Item        string
	ItemType    fs.FileMode
	Target      string
	TargetState *file.State
}

func (e *ConflictError) Error() string {
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
