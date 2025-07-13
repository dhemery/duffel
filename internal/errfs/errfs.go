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
	return &FS{root: newFile("", fs.ModeDir|0o775, "")}
}

// File is a file in a [FS] file system.
type File struct {
	info    fs.FileInfo
	dest    string           // The symlink destination if mode is symlink
	entries map[string]*File // The dir entries if mode is dir
	errors  map[Error]bool   // Errors to return from associated methods
}

func (fsys *FS) AddFile(name string, perm fs.FileMode, errs ...Error) error {
	return fsys.Add(name, perm.Perm(), "", errs...)
}

func (fsys *FS) AddDir(name string, perm fs.FileMode, errs ...Error) error {
	return fsys.Add(name, fs.ModeDir|perm.Perm(), "", errs...)
}

func (fsys *FS) AddLink(name string, dest string, errs ...Error) error {
	return fsys.Add(name, fs.ModeSymlink, dest, errs...)
}

func (fsys *FS) Add(name string, mode fs.FileMode, dest string, errs ...Error) error {
	f := newFile(path.Base(name), mode, dest, errs...)
	return fsys.add(path.Dir(name), f)
}

func (fsys *FS) add(dir string, f *File) error {
	const op = fsOp + "add"
	name := f.info.Name()
	fullName := path.Join(dir, name)

	if fullName == "." {
		return &fs.PathError{Op: op, Path: fullName, Err: fs.ErrInvalid}
	}

	parent, err := fsys.find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent = newFile(path.Base(dir), fs.ModeDir|0o755, "")
		err = fsys.add(path.Dir(dir), parent)
	}
	if err != nil {
		return &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.entries[name]; ok {
		return &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	parent.entries[f.Name()] = f

	return nil
}

type Error struct {
	s string
}

func (e Error) Error() string {
	return fsOp + e.s
}

func (fsys *FS) find(name string) (*File, error) {
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

// Open returnes the named file.
// If the file was created with ErrOpen,
// that error is returned instead.
func (fsys *FS) Open(name string) (fs.File, error) {
	const op = fsOp + "open"

	f, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}
	if f.errors[ErrOpen] {
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
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrLstat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrLstat}
	}

	return file.info, nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
// If the dir was created with ErrReadDir,
// that error is returned instead.
func (fsys *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	const op = fsOp + "readdir"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if !file.info.IsDir() {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if file.errors[ErrReadDir] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrReadDir}
	}

	var entries []fs.DirEntry
	for _, childName := range slices.Sorted(maps.Keys(file.entries)) {
		child := file.entries[childName]
		entry := fs.FileInfoToDirEntry(child.info)
		entries = append(entries, entry)
	}

	return entries, nil
}

// ReadLink returns the Dest of the named symlink file.
// If the file was created with ErrReadLink,
// that error is returned instead.
func (fsys *FS) ReadLink(name string) (string, error) {
	const op = fsOp + "readlink"
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

func (fsys *FS) Symlink(oldname, newname string) error {
	const op = fsOp + "symlink"
	dir := path.Dir(newname)
	base := path.Base(newname)
	return fsys.add(dir, newFile(base, fs.ModeSymlink, oldname))
}

// Stat returns a [fs.FileInfo] that describes the named file.
// This implementation of Stat does not follow symlinks.
// If the file was created with ErrStat,
// Lstat returns that error instead of the file info.
func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + "stat"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: name, Err: ErrStat}
	}

	return file.info, nil
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

func (f *File) Name() string {
	return f.info.Name()
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
	if f.errors[ErrStat] {
		return nil, &fs.PathError{Op: op, Path: f.Name(), Err: ErrStat}
	}
	return f.info, nil
}

func (f *File) String() string {
	return fmt.Sprintf("%q %s %q %v %v",
		f.info.Name(), f.info.Mode(), f.dest, f.entries, f.errors)
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
