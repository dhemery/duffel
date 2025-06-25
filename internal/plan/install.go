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

// Apply describes the installed state
// of the target file that corresponds to the given item.
// Pkg and item identify the item to be installed.
// Entry describes the state of the file in the source tree.
// TargetState describes the state of the target file
// after earlier tasks.
func (i Install) Apply(pkg, item string, entry fs.DirEntry, targetState *file.State) (*file.State, error) {
	pkgItem := path.Join(pkg, item)
	targetToItem := path.Join(i.TargetToSource, pkgItem)

	if targetState == nil {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return &file.State{Mode: fs.ModeSymlink, Dest: targetToItem}, err
	}

	if targetState.Mode.IsRegular() {
		return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrIsFile}
	}

	if targetState.Mode.IsDir() {
		return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrIsDir}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// InState is not file, dir, or link.
		return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrTargetType}
	}

	if targetState.Dest == targetToItem {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return targetState, err
	}

	return nil, &ErrConflict{Op: "install", Item: pkgItem, Err: ErrNotPkgItem}
}
