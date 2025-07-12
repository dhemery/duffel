// Package errfs provides a file system that can be customized
// to return specified errors.
package errfs

import (
	"errors"
	"io/fs"
	"path"
	"time"
)

const (
	errfsOp = "errfs."
)

type Error struct {
	s string
}

func (e Error) Error() string {
	return errfsOp + e.s
}

var (
	ErrLstat    = Error{"ErrLstat"}
	ErrOpen     = Error{"ErrOpen"}
	ErrReadDir  = Error{"ErrReadDir"}
	ErrReadLink = Error{"ErrReadLink"}
	ErrStat     = Error{"ErrStat"}
)

// File is a file in an [errfs] file system.
type File struct {
	name    string
	mode    fs.FileMode
	entries map[string]*File
	errors  map[Error]bool
	data    string // The symlink destination if mode has fs.ModeSymlink.
}

// FS is a tree of [File].
type FS struct {
	root *File
}

// New returns a new FS.
func New() FS {
	return FS{root: NewFile("", fs.ModeDir|0o775, "")}
}

// Create adds a file with the given name, mode, data, and prepared errors to FS.
// It also adds each ancestor directory that is not already in FS.
func (fsys FS) Create(name string, mode fs.FileMode, data string, errs ...Error) (*File, error) {
	const op = errfsOp + "create.file"
	base := path.Base(name)
	if base == "." {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	dir := path.Dir(name)
	parent, err := fsys.find(dir)

	if errors.Is(err, fs.ErrNotExist) {
		parent, err = fsys.Create(dir, fs.ModeDir|0o755, "")
	}
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.entries[base]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	f := NewFile(base, mode, data, errs...)

	parent.entries[f.name] = f

	return f, nil
}

func (fsys FS) find(name string) (*File, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.find(path.Dir(name))
	if err != nil {
		return nil, err
	}

	if !parent.IsDir() {
		return nil, fs.ErrInvalid
	}

	base := path.Base(name)
	for _, e := range parent.entries {
		if e.name == base {
			return e, nil
		}
	}
	return nil, fs.ErrNotExist
}

// Open always returns a [&fs.PathError].
func (fsys FS) Open(name string) (fs.File, error) {
	const op = errfsOp + "open"
	return nil, &fs.PathError{Op: op, Path: name, Err: errors.ErrUnsupported}
}

// Lstat returns a [fs.FileInfo] that describes the named file.
// Lstat does not follow symlinks.
// If the file was created with ErrLstat,
// Lstat returns that error instead of the file info.
func (fsys FS) Lstat(name string) (fs.FileInfo, error) {
	const op = errfsOp + "lstat"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrLstat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrLstat}
	}

	return file, nil
}

// ReadLink returns the Dest of the named symlink file.
// If the file was created with ErrReadLink,
// ReadLink returns that error of the Dest.
func (fsys FS) ReadLink(name string) (string, error) {
	const op = errfsOp + "readlink"
	file, err := fsys.find(name)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.errors[ErrReadLink] {
		return "", &fs.PathError{Op: op, Path: name, Err: ErrReadLink}
	}

	return file.data, nil
}

// NewFile returns a new [File]
// with the given name, mode, data, and prepared errors.
func NewFile(name string, mode fs.FileMode, data string, errs ...Error) *File {
	f := &File{
		name:    name,
		mode:    mode,
		data:    data,
		entries: map[string]*File{},
		errors:  map[Error]bool{},
	}
	for _, e := range errs {
		f.errors[e] = true
	}
	return f
}

func (t File) IsDir() bool {
	return t.mode&fs.ModeDir != 0
}

func (t File) ModTime() time.Time {
	return time.Time{}
}

func (t File) Mode() fs.FileMode {
	return t.mode
}

func (t File) Name() string {
	return t.name
}

func (t File) Size() int64 {
	return 0
}

func (t File) Sys() any {
	return nil
}
