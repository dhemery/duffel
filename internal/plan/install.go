package plan

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

type Merger interface {
	Merge(name string) error
}

func NewInstallOp(source, target string, merger Merger) installOp {
	targetToSource, err := filepath.Rel(target, source)
	if err != nil {
		panic(err)
	}
	return installOp{
		target:         target,
		targetToSource: targetToSource,
		merger:         merger,
	}
}

// installOp is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type installOp struct {
	target         string
	targetToSource string
	merger         Merger
}

// Apply describes the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the state of the target file
// after earlier tasks.
func (op installOp) Apply(pkg, item string, entry fs.DirEntry, targetState *file.State) (*file.State, error) {
	itemDepth := strings.Count(item, "/")
	itemToTarget := strings.Repeat("../", itemDepth)
	itemAsDest := path.Join(itemToTarget, op.targetToSource, pkg, item)

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
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
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
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
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

	if !targetState.DestMode.IsDir() {
		// Target links to a non-dir. Cannot merge with that.
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	if !entry.IsDir() {
		return nil, &InstallError{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetState: targetState,
		}
	}

	// The package item is a dir and the target is a link to a dir.
	// Merge the two by changeing the target to a dir,
	// then installing the contents of both the target's destination
	// and the package item into the new dir.
	//
	// First, install the contents of the target's destination,
	// as if the target were already a dir.
	mergePath := path.Join(op.target, targetState.Dest)
	if err := op.merger.Merge(mergePath); err != nil {
		return nil, err
	}

	// No conflicts installing the target destination's contents.
	// Now change the target to a dir, and walk the package item
	// to install its contents into the dir.
	dirState := &file.State{Mode: fs.ModeDir | 0o755}
	return dirState, nil
}

func (op installOp) toLinkDest(pkg, item string) string {
	itemDepth := strings.Count(item, "/")
	itemToTarget := strings.Repeat("../", itemDepth)
	return path.Join(itemToTarget, op.targetToSource, pkg, item)
}
