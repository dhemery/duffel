// Package errfs provides a file system that can be customized
// to return specified errors.
package errfs

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"time"
)

const (
	fsOp   = "errfs."
	fileOp = "errfs.file."
)

var (
	ErrClose    = Error{"ErrClose"}
	ErrLstat    = Error{"ErrLstat"}
	ErrOpen     = Error{"ErrOpen"}
	ErrRead     = Error{"ErrRead"}
	ErrReadDir  = Error{"ErrReadDir"}
	ErrReadLink = Error{"ErrReadLink"}
	ErrStat     = Error{"ErrStat"}
)

// FS is a tree of [*File].
type FS struct {
	root *File
}

// New returns a new FS.
func New() *FS {
	return &FS{root: NewDir("", 0o775)}
}

// File is a file in a [FS] file system.
type File struct {
	Info    fs.FileInfo
	Entries map[string]*File // The dir entries if mode is dir
	Dest    string           // The symlink destination if mode is symlink
	Errors  map[Error]bool
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
func (fsys *FS) Add(dir string, f *File) error {
	const op = fsOp + "add"
	name := f.Info.Name()
	fullName := path.Join(dir, name)

	if fullName == "." {
		return &fs.PathError{Op: op, Path: fullName, Err: fs.ErrInvalid}
	}

	parent, err := fsys.Find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent = NewDir(path.Base(dir), 0o755)
		err = fsys.Add(path.Dir(dir), parent)
	}
	if err != nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.Entries[name]; ok {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	parent.Entries[f.Name()] = f

	return nil
}

type Error struct {
	s string
}

func (e Error) Error() string {
	return fsOp + e.s
}

func (fsys *FS) Find(name string) (*File, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.Find(path.Dir(name))
	if err != nil {
		return nil, err
	}

	if !parent.Info.IsDir() {
		return nil, fs.ErrInvalid
	}

	base := path.Base(name)
	for _, e := range parent.Entries {
		if e.Info.Name() == base {
			return e, nil
		}
	}
	return nil, fs.ErrNotExist
}

// Open returnes the named file.
// If the file was created with ErrOpen,
// that error is returned instead.
func (fsys *FS) Open(name string) (fs.File, error) {
	const op = fsOp + "open"

	f, err := fsys.Find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	if f.Errors[ErrOpen] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrOpen}
	}
	return f, nil
}

// Lstat returns a [fs.FileInfo] that describes the named file.
// Lstat does not follow symlinks.
// If the file was created with ErrLstat,
// that error is returned instead.
func (fsys *FS) Lstat(name string) (fs.FileInfo, error) {
	const op = fsOp + "lstat"
	file, err := fsys.Find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.Errors[ErrLstat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrLstat}
	}

	return file.Info, nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
// If the dir was created with ErrReadDir,
// that error is returned instead.
func (fsys *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	const op = fsOp + "readdir"
	file, err := fsys.Find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if !file.Info.IsDir() {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.Errors[ErrReadDir] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrReadDir}
	}

	var entries []fs.DirEntry
	for _, childName := range slices.Sorted(maps.Keys(file.Entries)) {
		child := file.Entries[childName]
		entry := fs.FileInfoToDirEntry(child.Info)
		entries = append(entries, entry)
	}

	return entries, nil
}

// ReadLink returns the Dest of the named symlink file.
// If the file was created with ErrReadLink,
// that error is returned instead.
func (fsys *FS) ReadLink(name string) (string, error) {
	const op = fsOp + "readlink"
	file, err := fsys.Find(name)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.Info.Mode()&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.Errors[ErrReadLink] {
		return "", &fs.PathError{Op: op, Path: name, Err: ErrReadLink}
	}

	return file.Dest, nil
}

func (fsys *FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	dir := path.Dir(newname)
	base := path.Base(newname)
	return fsys.Add(dir, NewSymlink(base, oldname))
}

// Stat returns a [fs.FileInfo] that describes the named file.
// This implementation of Stat does not follow symlinks.
// If the file was created with ErrStat,
// Lstat returns that error instead of the file info.
func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	file, err := fsys.Find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.Errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrStat}
	}

	return file.Info, nil
}

func newFile(name string, mode fs.FileMode, data string, errs ...Error) *File {
	f := &File{
		Info:    &info{N: name, M: mode},
		Dest:    data,
		Entries: map[string]*File{},
		Errors:  map[Error]bool{},
	}
	for _, e := range errs {
		f.Errors[e] = true
	}
	return f
}

func (f *File) Name() string {
	return f.Info.Name()
}

func (f *File) Close() error {
	const op = fileOp + "close"
	return &fs.PathError{Op: op, Path: f.Name(), Err: errors.ErrUnsupported}
}

func (f *File) Read([]byte) (int, error) {
	const op = fileOp + "read"
	return 0, &fs.PathError{Op: op, Path: f.Name(), Err: errors.ErrUnsupported}
}

func (f *File) Stat() (fs.FileInfo, error) {
	const op = fileOp + "stat"
	if f.Errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: f.Name(), Err: ErrStat}
	}
	return f.Info, nil
}

func (f *File) String() string {
	return fmt.Sprintf("%q %s %q %v %v",
		f.Info.Name(), f.Info.Mode(), f.Dest, f.Entries, f.Errors)
}

type info struct {
	N string
	M fs.FileMode
}

func (i *info) IsDir() bool {
	return i.M&fs.ModeDir != 0
}

func (i *info) ModTime() time.Time {
	return time.Time{}
}

func (i *info) Mode() fs.FileMode {
	return i.M
}

func (i *info) Name() string {
	return i.N
}

func (i *info) Size() int64 {
	return 0
}

func (i *info) Sys() any {
	return nil
}
