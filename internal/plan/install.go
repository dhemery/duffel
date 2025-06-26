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
	ErrDestNotPkgItem = errors.New("destination is not a package item")
	ErrNotDir         = errors.New("is not a directory")
	ErrIsDir          = errors.New("is a directory")
	ErrIsFile         = errors.New("is a file")
	ErrUnknownType    = errors.New("is not a file, dir, or link")
)

type ErrInvalidTarget struct {
	Op    string
	Pkg   string
	Item  string
	State *file.State
	Err   error
}

func (e *ErrInvalidTarget) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("cannot %s source %s: existing target %s (%s) %s",
		e.Op, pkgItem, e.Item, stateString(e.State), e.Err)
}

func stateString(s *file.State) string {
	result := typeString(s.Mode)
	if s.Dest != "" {
		result += " " + s.Dest
	}
	return result
}

func (e *ErrInvalidTarget) Unwrap() error { return e.Err }

type ErrConflict struct {
	Op          string
	Pkg         string
	Item        string
	SourceType  fs.FileMode
	TargetState *file.State
}

func typeString(t fs.FileMode) string {
	switch {
	case t.IsRegular():
		return "regular file"
	case t.IsDir():
		return "directory"
	case t&fs.ModeSymlink != 0:
		return "symlink"
	default:
		return "unknown file type " + t.String()
	}
}

func (e *ErrConflict) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s cannot replace or merge target %s (%s) with source %s (%s)",
		e.Op, e.Item, stateString(e.TargetState), pkgItem, typeString(e.SourceType))
}

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
		return nil, &ErrInvalidTarget{
			Op: "install", Pkg: pkg, Item: item,
			State: targetState, Err: ErrIsFile,
		}
	}

	if targetState.Mode.IsDir() {
		// If target and pkg item are both dirs, install the pkg item's contents
		if entry.IsDir() {
			return targetState, nil
		}
		return nil, &ErrConflict{
			Op: "install", Pkg: pkg, Item: item,
			SourceType: entry.Type(), TargetState: targetState,
		}
	}

	if targetState.Mode.Type() != fs.ModeSymlink {
		// Target item is not file, dir, or link.
		return nil, &ErrInvalidTarget{
			Op: "install", Pkg: pkg, Item: item,
			State: targetState, Err: ErrUnknownType,
		}
	}

	if targetState.Dest == itemAsDest {
		var err error
		if entry.IsDir() {
			err = fs.SkipDir
		}
		return targetState, err
	}

	if !entry.IsDir() {
		return nil, &ErrConflict{
			Op: "install", Pkg: pkg, Item: item,
			SourceType: entry.Type(), TargetState: targetState,
		}
	}

	return nil, &ErrInvalidTarget{
		Op: "install", Pkg: pkg, Item: item,
		State: targetState, Err: ErrDestNotPkgItem,
	}
}

func (i Install) toLinkDest(pkgItem string) string {
	depth := strings.Count(pkgItem, "/") - 1
	prefix := strings.Repeat("../", depth)
	return path.Join(prefix, i.TargetToSource, pkgItem)
}
