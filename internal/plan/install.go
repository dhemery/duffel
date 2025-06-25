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
	Pkg  string
	Item string
	Dest string
	Err  error
}

func (e *ErrTargetDest) Error() string {
	return fmt.Sprintf("%s package %s item %s existing target link destination %s: %s",
		e.Op, e.Pkg, e.Item, e.Dest, e.Err)
}

func (e *ErrTargetDest) Unwrap() error { return e.Err }

type ErrTargetType struct {
	Op   string
	Pkg  string
	Item string
	Type fs.FileMode
	Err  error
}

func (e *ErrTargetType) Error() string {
	return fmt.Sprintf("%s package %s item %s existing target (%s): %s",
		e.Op, e.Pkg, e.Item, e.Type, e.Err)
}

func (e *ErrTargetType) Unwrap() error { return e.Err }

type ErrConflict struct {
	Op         string
	Pkg        string
	Item       string
	ItemType   fs.FileMode
	TargetMode fs.FileMode
	Err        error
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("%s cannot replace/merge target %s (%s) with package %s item %s (%s): %s",
		e.Op, e.Item, e.TargetMode, e.Pkg, e.Item, e.ItemType, e.Err)
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
	itemAsDest := i.toLinkDest(pkgItem)

	if targetState == nil {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return &file.State{Mode: fs.ModeSymlink, Dest: itemAsDest}, err
	}

	if targetState.Mode.IsRegular() {
		return nil, &ErrTargetType{
			Op: "install", Pkg: pkg, Item: item,
			Type: targetState.Mode.Type(), Err: ErrIsFile,
		}
	}

	if targetState.Mode.IsDir() {
		// If target and pkg item are both dirs, install the pkg item's contents
		if entry.IsDir() {
			return targetState, nil
		}
		return nil, &ErrConflict{
			Op: "install", Pkg: pkg, Item: item,
			ItemType: entry.Type(), TargetMode: targetState.Mode, Err: ErrIsDir,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &ErrTargetType{
			Op: "install", Pkg: pkg, Item: item,
			Type: targetState.Mode.Type(), Err: ErrUnknownType,
		}
	}

	if targetState.Dest == itemAsDest {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return targetState, err
	}

	return nil, &ErrTargetDest{
		Op: "install", Pkg: pkg, Item: item,
		Dest: targetState.Dest, Err: ErrNotPkgItem,
	}
}

func (i Install) toLinkDest(pkgItem string) string {
	depth := strings.Count(pkgItem, "/") - 1
	prefix := strings.Repeat("../", depth)
	return path.Join(prefix, i.TargetToSource, pkgItem)
}
