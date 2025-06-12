package duftest

import (
	"cmp"
	"errors"
	"io/fs"
)

type ErrFS struct {
	OpenErr error
	StatErr error
}

func (f ErrFS) Open(path string) (fs.File, error) {
	return nil, cmp.Or(f.OpenErr, errors.ErrUnsupported)
}

func (f ErrFS) Stat(path string) (fs.FileInfo, error) {
	return nil, cmp.Or(f.StatErr, errors.ErrUnsupported)
}
