package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
)

var (
	ErrDestNotPkgItem = errors.New("destination is not a package item")
	ErrIsDir          = errors.New("is a directory")
	ErrIsFile         = errors.New("is a file")
	ErrUnknownType    = errors.New("is not a file, dir, or link")
)

type TargetError struct {
	Op   string
	Pkg  string
	Item string
	Type fs.FileMode
	Dest string
	Err  error
}

func (e *TargetError) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s %q: cannot alter target %q (%s): %s",
		e.Op, pkgItem, e.Item, typeString(e.Type), e.Err)
}

func (e *TargetError) Unwrap() error { return e.Err }

type ConflictError struct {
	Op         string
	Pkg        string
	Item       string
	ItemType   fs.FileMode
	TargetType fs.FileMode
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

func (e *ConflictError) Error() string {
	pkgItem := path.Join(e.Pkg, e.Item)
	return fmt.Sprintf("%s %q: cannot replace or merge target %q with package item %q: target is a %s, package item is a %s",
		e.Op, pkgItem, e.Item, pkgItem, typeString(e.TargetType), typeString(e.ItemType))
}
