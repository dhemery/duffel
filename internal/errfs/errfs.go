// Package errfs provides a file system that can be customized
// to return specified errors.
package errfs

import (
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"time"
)

const (
	lstatOp    = "lstat"
	openOp     = "open"
	readOp     = "read"
	readDirOp  = "readdir"
	readLinkOp = "readlink"
	statOp     = "stat"
	symlinkOp  = "symlink" // For Error, use writeOp.

	fsOp   = "errfs."      // Prefix added FS ops in error messages.
	fileOp = "errfs.file." // Prefix added to File ops in error messages.
	addOp  = "add"         // For the Add helper methods.
)

// New returns a new FS.
func New() *FS {
	return &FS{root: newNode(NewDir("", 0o755))}
}

type node struct {
	file    *File
	entries map[string]node
}

func newNode(f *File) node {
	return node{f, map[string]node{}}
}

// FS is a limited, in-memory [fs.FS] that can be configured
// to return specified errors from operations on the file system and its files.
type FS struct {
	root node
}

// add adds a node for f to fsys and returns the node.
func (fsys *FS) add(file *File) (node, error) {
	const op = fsOp + addOp
	node := newNode(file)

	name := file.Name
	if name == "." {
		return node, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	dir := path.Dir(name)
	parent, err := fsys.find(dir)
	if errors.Is(err, fs.ErrNotExist) {
		parent, err = fsys.add(NewDir(dir, 0o755))
	}
	if err != nil {
		return node, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.entries[name]; ok {
		return node, &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	parent.entries[name] = node

	return node, nil
}

// find returns the node for the named file.
func (fsys *FS) find(name string) (node, error) {
	if name == "." {
		return fsys.root, nil
	}

	parent, err := fsys.find(path.Dir(name))
	if err != nil {
		return node{}, err
	}

	if !parent.file.Mode.IsDir() {
		return node{}, fs.ErrInvalid
	}

	for _, entry := range parent.entries {
		if entry.file.Name == name {
			return entry, nil
		}
	}
	return node{}, fs.ErrNotExist
}

// Open returnes the named file.
// If the file was created with an Open [Error],
// that error is returned instead.
func (fsys *FS) Open(name string) (fs.File, error) {
	const op = fsOp + openOp

	node, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	file := node.file
	if opErr, ok := file.errors[openOp]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: opErr}
	}
	return file, nil
}

// Lstat returns a [fs.FileInfo] that describes the named file.
// Lstat does not follow symlinks.
// If the file was created with an Lstat [Error],
// that error is returned instead.
func (fsys *FS) Lstat(name string) (fs.FileInfo, error) {
	const op = fsOp + lstatOp
	node, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	file := node.file
	if opErr, ok := file.errors[lstatOp]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: opErr}
	}

	return fileInfo{file}, nil
}

// ReadDir reads the named directory
// and returns a list of directory entries sorted by filename.
// If the directory was created with a ReadDir [Error],
// that error is returned instead.
func (fsys *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	const op = fsOp + readDirOp
	node, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	file := node.file
	if !file.Mode.IsDir() {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if opErr, ok := file.errors[readDirOp]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: opErr}
	}

	var entries []fs.DirEntry
	for _, key := range slices.Sorted(maps.Keys(node.entries)) {
		entries = append(entries, dirEntry{node.entries[key].file})
	}

	return entries, nil
}

// ReadLink returns the Dest of the named symlink file.
// If the file was created with a ReadLink [Error],
// that error is returned instead.
func (fsys *FS) ReadLink(name string) (string, error) {
	const op = fsOp + readLinkOp
	node, err := fsys.find(name)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}

	file := node.file
	if file.Mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	if opErr, ok := file.errors[readLinkOp]; ok {
		return "", &fs.PathError{Op: op, Path: name, Err: opErr}
	}

	return file.Dest, nil
}

// Symlink creates a new symlink with the given name and destination.
// TODO: Do not use fsys.add. Instead find the parent, and fail if error.
// TODO: Return an error if the parent was created with a Symlink.
func (fsys *FS) Symlink(dest, name string) error {
	const op = fsOp + symlinkOp
	if err := Add(fsys, NewLink(name, dest)); err != nil {
		return &os.LinkError{Op: op, Old: dest, New: name, Err: err}
	}
	return nil
}

// Stat returns a [fs.FileInfo] that describes the named file.
// This implementation of Stat does not follow symlinks.
// If the file was created with a Stat [Error],
// that error is returned instead.
func (fsys *FS) Stat(name string) (fs.FileInfo, error) {
	const op = fsOp + statOp
	node, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	file := node.file
	if opErr, ok := file.errors[statOp]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: opErr}
	}

	return fileInfo{file}, nil
}

func (fsys *FS) String() string {
	var out strings.Builder
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		file, err := Find(fsys, name)
		if err != nil {
			return err
		}
		out.WriteRune('"')
		out.WriteString(name)
		out.WriteString(`" `)
		out.WriteString(file.String())
		out.WriteRune('\n')
		return nil
	})
	if err != nil {
		return err.Error()
	}
	return out.String()
}

type File struct {
	Name   string           // The full name of the file.
	Mode   fs.FileMode      // The file mode.
	Dest   string           // The link destination if the file is a symlink.
	errors map[string]Error // Errors to return from relevant operations.
}

func newFile(name string, mode fs.FileMode, dest string, errs ...Error) *File {
	f := &File{
		Name:   name,
		Mode:   mode,
		Dest:   dest,
		errors: map[string]Error{},
	}
	for _, e := range errs {
		f.errors[e.op] = e
	}
	return f
}

// Close implements fs.File.
// It does nothing.
func (f *File) Close() error {
	return nil
}

// Read implements fs.File.
// It always returns 0, [errors.ErrUnsupported].
func (f *File) Read([]byte) (int, error) {
	const op = fileOp + readOp
	return 0, &fs.PathError{Op: op, Path: f.Name, Err: errors.ErrUnsupported}
}

// Stat returns a [fs.FileInfo] that describes f.
// This implementation of Stat does not follow symlinks.
// If the file was created with a Stat [Error],
// that error is returned instead.
func (f *File) Stat() (fs.FileInfo, error) {
	const op = fileOp + statOp
	if opErr, ok := f.errors[statOp]; ok {
		return nil, &fs.PathError{Op: op, Path: f.Name, Err: opErr}
	}
	return fileInfo{f}, nil
}

func (f *File) String() string {
	var out strings.Builder
	out.WriteString(f.Mode.String())
	if f.Mode&fs.ModeSymlink != 0 {
		out.WriteString(` "`)
		out.WriteString(f.Dest)
		out.WriteRune('"')
	}
	var errors []string
	for _, err := range f.errors {
		errors = append(errors, err.Error())
	}
	if len(errors) > 1 {
		out.WriteString(" [")
		out.WriteString(strings.Join(errors, ", "))
		out.WriteRune(']')
	}
	return out.String()
}

type fileInfo struct {
	file *File
}

// IsDir implements fs.DirEntry and fs.FileInfo.
func (i fileInfo) IsDir() bool {
	return i.Mode().IsDir()
}

// ModTime implements fs.FileInfo.
// It always returns the zero time.
func (i fileInfo) ModTime() time.Time {
	return time.Time{}
}

// Mode implements fs.FileInfo.
func (i fileInfo) Mode() fs.FileMode {
	return i.file.Mode
}

// Name implements fs.FileInfo.
func (i fileInfo) Name() string {
	return path.Base(i.file.Name)
}

// Size implements fs.FileInfo.
// It always returns 0.
func (i fileInfo) Size() int64 {
	return 0
}

// Sys implements fs.FileInfo.
// It returns the full name of the file.
func (i fileInfo) Sys() any {
	return i.file.Name
}

type dirEntry struct {
	file *File
}

// Info implements fs.DirEntry.
func (e dirEntry) Info() (fs.FileInfo, error) {
	return fileInfo(e), nil
}

// IsDir implements fs.DirEntry.
func (e dirEntry) IsDir() bool {
	return e.Type().IsDir()
}

// Name implements fs.DirEntry.
func (e dirEntry) Name() string {
	return path.Base(e.file.Name)
}

// Type implements fs.DirEntry.
func (e dirEntry) Type() fs.FileMode {
	return e.file.Mode.Type()
}
