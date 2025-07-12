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

var (
	ErrLstat    = Error{"ErrLstat"}
	ErrOpen     = Error{"ErrOpen"}
	ErrReadDir  = Error{"ErrReadDir"}
	ErrReadLink = Error{"ErrReadLink"}
	ErrStat     = Error{"ErrStat"}
)

// FS is a tree of [*File].
type FS struct {
	root *File
}

// New returns a new FS.
func New() FS {
	return FS{root: NewDir("", 0o775)}
}

// File is a file in a [FS] file system.
type File struct {
	info    fs.FileInfo
	entries map[string]*File // The dir entries if mode is dir
	dest    string           // The symlink destination if mode is symlink
	errors  map[Error]bool
}

// NewDir returns a new directory *File
// with the given name, mode, and prepared errors.
func NewDir(name string, mode fs.FileMode, errs ...Error) *File {
	return newFile(name, mode|fs.ModeDir, "", errs...)
}

// NewSymlink returns a new symlink *File
// with the given name, destination, and prepared errors.
func NewSymlink(name, dest string, errs ...Error) *File {
	return newFile(name, fs.ModeSymlink, dest, errs...)
}

// NewFile returns a new *File
// with the given name, mode, and prepared errors.
func NewFile(name string, mode fs.FileMode, errs ...Error) *File {
	return newFile(name, mode, "", errs...)
}

// Add adds f to directory dir in fsys.
// Dir and its ancestors are also added
// if they do not yet exist in fsys.
func (fsys FS) Add(dir string, f *File) error {
	const op = errfsOp + "add"
	name := f.info.Name()
	fullName := path.Join(dir, name)

	if fullName == "." {
		return &fs.PathError{Op: op, Path: fullName, Err: fs.ErrInvalid}
	}

	parent, err := fsys.find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent = NewDir(path.Base(dir), 0o755)
		err = fsys.Add(path.Dir(dir), parent)
	}
	if err != nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.entries[name]; ok {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	parent.entries[f.info.Name()] = f

	return nil
}

type Error struct {
	s string
}

func (e Error) Error() string {
	return errfsOp + e.s
}

func (fsys FS) find(name string) (*File, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.find(path.Dir(name))
	if err != nil {
		return nil, err
	}

	if !parent.info.IsDir() {
		return nil, fs.ErrInvalid
	}

	base := path.Base(name)
	for _, e := range parent.entries {
		if e.info.Name() == base {
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

	return file.info, nil
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

	if file.info.Mode()&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.errors[ErrReadLink] {
		return "", &fs.PathError{Op: op, Path: name, Err: ErrReadLink}
	}

	return file.dest, nil
}

func newFile(name string, mode fs.FileMode, data string, errs ...Error) *File {
	f := &File{
		info:    &info{name: name, mode: mode},
		dest:    data,
		entries: map[string]*File{},
		errors:  map[Error]bool{},
	}
	for _, e := range errs {
		f.errors[e] = true
	}
	return f
}

type info struct {
	name string
	mode fs.FileMode
}

func (i *info) IsDir() bool {
	return i.mode&fs.ModeDir != 0
}

func (i *info) ModTime() time.Time {
	return time.Time{}
}

func (i *info) Mode() fs.FileMode {
	return i.mode
}

func (i *info) Name() string {
	return i.name
}

func (i *info) Size() int64 {
	return 0
}

func (i *info) Sys() any {
	return nil
}
