package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/dhemery/duffel/internal/file"
)

var (
	ErrNotPkgItem = errors.New("destination is not a package item")
	ErrIsDir      = errors.New("is a directory")
	ErrIsFile     = errors.New("is a file")
	ErrTargetType = errors.New("is not file, dir, or link")
)

type ErrConflict struct {
	Op   string
	Item string
	Err  error
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("%s item %s conflicts with target: %s", e.Op, e.Item, e.Err)
}

func (e *ErrConflict) Unwrap() error { return e.Err }

// Install is an [ItemOp] that describes the installed states
// of the target files that correspond to the given pkg items.
type Install struct {
	TargetToSource string // The relative path from the target dir to the source dir.
}

// Apply returns a State describing the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// InState describes the desired state of the target file
// as determined by prior analysis.
func (i Install) Apply(pkg, item string, entry fs.DirEntry, inState *file.State) (*file.State, error) {
	pkgItem := path.Join(pkg, item)
	targetToItem := path.Join(i.TargetToSource, pkgItem)

	if inState == nil {
		// No conflicting target state. Link to this pkg item.
		return &file.State{Mode: fs.ModeSymlink, Dest: targetToItem}, nil
	}

	if inState.Mode.IsRegular() {
		return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrIsFile}
	}

	if inState.Mode.IsDir() {
		return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrIsDir}
	}

	if inState.Mode.Type() != fs.ModeSymlink {
		// InState is not file, dir, or link.
		return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrTargetType}
	}

	if inState.Dest == targetToItem {
		// Target already links to this pkg item.
		return inState, nil
	}

	return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrNotPkgItem}
}
