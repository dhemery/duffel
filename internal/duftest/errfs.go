package duftest

import (
	"errors"
	"io/fs"
	"path"
	"time"
)

const (
	testfsOp = "testfs."
)

type TestFS struct {
	root *TestFile
}

func NewTestFS() TestFS {
	return TestFS{root: NewTestFile("", fs.ModeDir|0o775)}
}

func (fsys TestFS) Create(name string, mode fs.FileMode) (*TestFile, error) {
	const op = testfsOp + "create.file"
	base := path.Base(name)
	if base == "." {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	dir := path.Dir(name)
	parent, err := fsys.find(dir)

	if errors.Is(err, fs.ErrNotExist) {
		parent, err = fsys.Create(dir, fs.ModeDir|0o755)
	}
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if _, ok := parent.entries[base]; ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrExist}
	}

	f := NewTestFile(base, mode)

	parent.entries[f.name] = f

	return f, nil
}

func (fsys TestFS) find(name string) (*TestFile, error) {
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

func (fsys TestFS) Open(name string) (fs.File, error) {
	const op = testfsOp + "open"
	return nil, &fs.PathError{Op: op, Path: name, Err: errors.ErrUnsupported}
}

func (fsys TestFS) Lstat(name string) (fs.FileInfo, error) {
	const op = testfsOp + "lstat"
	file, err := fsys.find(name)
	if err != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: err}
	}

	if file.LstatErr != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: file.LstatErr}
	}

	return file, nil
}

func (fsys TestFS) ReadLink(name string) (string, error) {
	const op = testfsOp + "readlink"
	file, err := fsys.find(name)
	if err != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: err}
	}
	if file.ReadLinkErr != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: file.ReadLinkErr}
	}

	if file.mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	return file.Dest, nil
}

func NewTestFile(name string, mode fs.FileMode) *TestFile {
	return &TestFile{
		name:    name,
		mode:    mode,
		entries: map[string]*TestFile{},
	}
}

type TestFile struct {
	name        string
	mode        fs.FileMode
	entries     map[string]*TestFile
	Dest        string
	LstatErr    error
	OpenErr     error
	ReadDirErr  error
	ReadLinkErr error
	StatErr     error
}

func (t TestFile) IsDir() bool {
	return t.mode&fs.ModeDir != 0
}

func (t TestFile) ModTime() time.Time {
	return time.Time{}
}

func (t TestFile) Mode() fs.FileMode {
	return t.mode
}

func (t TestFile) Name() string {
	return t.name
}

func (t TestFile) Size() int64 {
	return 0
}

func (t TestFile) Sys() any {
	return nil
}
