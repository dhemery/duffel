package rules

import (
	"io/fs"
	"testing/fstest"
	"time"
)

func directory(mode fs.FileMode) *fstest.MapFile {
	return &fstest.MapFile{Mode: mode | fs.ModeDir}
}

func regularFile() *fstest.MapFile {
	return &fstest.MapFile{Mode: 0644}
}

func linkTo(p string) *fstest.MapFile {
	return &fstest.MapFile{
		Mode: 0644 | fs.ModeSymlink,
		Data: []byte(p),
	}
}

type dirEntry struct {
	name string
	file *fstest.MapFile
}

func (e dirEntry) Info() (fs.FileInfo, error) {
	return e, nil
}

func (e dirEntry) Mode() fs.FileMode {
	return e.file.Mode
}

func (e dirEntry) IsDir() bool {
	return e.file.Mode.IsDir()
}

func (dirEntry) ModTime() time.Time {
	panic("unimplemented")
}

func (e dirEntry) Name() string {
	return e.name
}

func (e dirEntry) Size() int64 {
	return int64(len(e.file.Data))
}

func (dirEntry) Sys() any {
	return nil
}

func (e dirEntry) Type() fs.FileMode {
	return e.Mode()
}
