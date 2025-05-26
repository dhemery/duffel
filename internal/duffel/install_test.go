package duffel

import (
	"bytes"
	"errors"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestInstallPackageSolo(t *testing.T) {
	req := &Request{
		FS:     newTestFS(),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Source: "home/user/source",
		Target: "home/user",
	}
	Install(req)
}

func newTestFS() FS {
	return testFS{root: "testfs"}
}

type testFS struct {
	fstest.MapFS
	root string
}

func (f testFS) Symlink(old string, new string) error {
	return errors.New("not implemented")
}

func (f testFS) Join(path string) string {
	return filepath.Join(f.root, path)
}
