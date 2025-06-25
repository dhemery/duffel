package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/dhemery/duffel/internal/file"
)

var (
	ErrNotPkgItem  = errors.New("not a package item")
	ErrIsDir       = errors.New("is a directory")
	ErrIsFile      = errors.New("is a file")
	ErrUnknownType = errors.New("not file, dir, or link")
)

type ErrTargetDest struct {
	Op   string
	Item string
	Dest string
	Err  error
}

func (e *ErrTargetDest) Error() string {
	return fmt.Sprintf("%s target %s destination %s: %s",
		e.Op, e.Item, e.Dest, e.Err)
}

func (e *ErrTargetDest) Unwrap() error { return e.Err }

type ErrTargetType struct {
	Op   string
	Item string
	Mode fs.FileMode
	Err  error
}

func (e *ErrTargetType) Error() string {
	return fmt.Sprintf("%s target %s (%s): %s",
		e.Op, e.Item, e.Mode, e.Err)
}

func (e *ErrTargetType) Unwrap() error { return e.Err }

type ErrConflict struct {
	Op         string
	Item       string
	ItemType   fs.FileMode
	TargetMode fs.FileMode
	Err        error
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("%s cannot replace/merge target %s (%s) with pkg item (%s): %s",
		e.Op, e.Item, e.TargetMode, e.ItemType, e.Err)
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
	depth := strings.Count(item, "/")
	prefix := strings.Repeat("../", depth)
	targetToItem := path.Join(prefix, i.TargetToSource, pkgItem)

	if targetState == nil {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return &file.State{Mode: fs.ModeSymlink, Dest: targetToItem}, err
	}

	if targetState.Mode.IsRegular() {
		return nil, &ErrTargetType{
			Op:   "install",
			Item: pkgItem, Mode: targetState.Mode, Err: ErrIsFile,
		}
	}

	if targetState.Mode.IsDir() {
		// If target and pkg item are both dirs, install the pkg item's contents
		if entry.IsDir() {
			return targetState, nil
		}
		return nil, &ErrConflict{
			Op:   "install",
			Item: pkgItem, ItemType: entry.Type(), TargetMode: targetState.Mode, Err: ErrIsDir,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &ErrTargetType{
			Op:   "install",
			Item: pkgItem, Mode: targetState.Mode, Err: ErrUnknownType,
		}
	}

	if targetState.Dest == targetToItem {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return targetState, err
	}

	return nil, &ErrTargetDest{
		Op:   "install",
		Item: pkgItem, Dest: targetState.Dest, Err: ErrNotPkgItem,
	}
}
