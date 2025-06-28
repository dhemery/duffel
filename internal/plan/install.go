package plan

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

// Install is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type Install struct {
	Source string
	Target string
}

// Apply describes the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the state of the target file
// after earlier tasks.
func (i Install) Apply(pkg, item string, entry fs.DirEntry, targetState *file.State) (*file.State, error) {
	itemAsDest := i.toLinkDest(pkg, item)

	if targetState == nil {
		// There is no target item, so we're free to create a link to the pkg item.
		var err error
		if entry.IsDir() {
			// Linking to the dir installs the dir and its contents.
			// There's no need to walk its contents.
			err = fs.SkipDir
		}
		return &file.State{Mode: fs.ModeSymlink, Dest: itemAsDest}, err
	}

	// At this point, we know that the target exists,
	// either on the file system or as planned by a previous operation.

	if targetState.Mode.IsRegular() {
		// Cannot modify an existing regular file.
		return nil, &TargetError{
			Op: "install", Pkg: pkg, Item: item,
			State: targetState,
		}
	}

	if targetState.Mode.IsDir() {
		if entry.IsDir() {
			// The target and pkg item are both dirs.
			// Return the target state unchanged,
			// and a nil error to walk the pkg item's contents.
			return targetState, nil
		}

		// The target is a dir, but the pkg item is not.
		// Cannot merge the target dir with a non-dir pkg item.
		return nil, &ConflictError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &TargetError{
			Op: "install", Pkg: pkg, Item: item,
			State: targetState,
		}
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

	if !entry.IsDir() {
		return nil, &ConflictError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	return nil, &TargetError{
		Op: "install", Pkg: pkg, Item: item,
		State: targetState,
	}
}

func (i Install) toLinkDest(pkg, item string) string {
	targetToSource, err := filepath.Rel(i.Target, i.Source)
	if err != nil {
		panic(err)
	}
	itemDepth := strings.Count(item, "/")
	itemToTarget := strings.Repeat("../", itemDepth)
	return path.Join(itemToTarget, targetToSource, pkg, item)
}
