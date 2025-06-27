package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

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
	Op   string
	Pkg  string
	Item string
	Type fs.FileMode
	Dest string
	Err  error
}

func (e *ErrInvalidTarget) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s %s cannot alter target %s (%s): %s",
		e.Op, pkgItem, e.Item, typeString(e.Type), e.Err)
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
