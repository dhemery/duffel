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

type TestFS map[string]TestFile

func (fsys TestFS) Open(name string) (fs.File, error) {
	const op = testfsOp + "open"
	return nil, &fs.PathError{Op: op, Path: name, Err: errors.ErrUnsupported}
}

func (fsys TestFS) Lstat(name string) (fs.FileInfo, error) {
	const op = testfsOp + "lstat"
	file, ok := fsys[name]
	if !ok {
		return nil, &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
	}

	if file.LstatErr != nil {
		return nil, &fs.PathError{Op: op, Path: name, Err: file.LstatErr}
	}

	return TestFileInfo{name: path.Base(name), mode: file.Mode}, nil
}

func (fsys TestFS) ReadLink(name string) (string, error) {
	const op = testfsOp + "readlink"
	file, ok := fsys[name]
	if !ok {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
	}
	if file.ReadLinkErr != nil {
		return "", &fs.PathError{Op: op, Path: name, Err: file.ReadLinkErr}
	}

	if file.Mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}

	return file.Dest, nil
}

type TestFile struct {
	Mode        fs.FileMode
	Dest        string
	LstatErr    error
	ReadLinkErr error
}

type TestFileInfo struct {
	name string
	mode fs.FileMode
}

func (t TestFileInfo) IsDir() bool {
	return t.Mode()&fs.ModeDir != 0
}

func (t TestFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (t TestFileInfo) Mode() fs.FileMode {
	return t.mode
}

func (t TestFileInfo) Name() string {
	return t.name
}

func (t TestFileInfo) Size() int64 {
	return 0
}

func (t TestFileInfo) Sys() any {
	return nil
}
