package plan

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

type Merger interface {
	Merge(name, target string) error
}

func NewInstallOp(source, target string, merger Merger) installOp {
	return installOp{
		target: target,
		source: source,
		merger: merger,
	}
}

// installOp is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type installOp struct {
	source string
	target string
	merger Merger
}

// Apply describes the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the state of the target file
// after earlier tasks.
func (op installOp) Apply(name string, entry fs.DirEntry, targetState *State) (*State, error) {
	pkgItem := name[len(op.source)+1:]
	_, item, _ := strings.Cut(pkgItem, "/")
	targetItem := path.Join(op.target, item)
	itemAsDest, _ := filepath.Rel(path.Dir(targetItem), name)

	if targetState == nil {
		// There is no target item, so we're free to create a link to the pkg item.
		var err error
		if entry.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return &State{
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
	dirState := &State{Type: fs.ModeDir}
	return dirState, nil
}

func conflictError(item string, entry fs.DirEntry, target string, state *State) error {
	return &InstallError{
		Item:        item,
		ItemType:    entry.Type(),
		Target:      target,
		TargetState: state,
	}
}
