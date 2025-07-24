package duftest

import (
	"io/fs"
	"os"
	"testing"
)

// Must returns a helper whose methods wrap file system functions
// and report errors to t as fatal errors.
func Must(t *testing.T) must {
	return must{t}
}

type must struct {
	*testing.T
}

func (m must) Lstat(name string) fs.FileInfo {
	m.Helper()
	e, err := os.Lstat(name)
	if err != nil {
		m.Fatal("must lstat", err)
	}
	return e
}

func (m must) MkdirAll(name string, perm fs.FileMode) {
	m.Helper()
	if err := os.MkdirAll(name, perm); err != nil {
		m.Fatal("must mkdir all", err)
	}
}

func (m must) ReadDir(name string) []fs.DirEntry {
	m.Helper()
	ee, err := os.ReadDir(name)
	if err != nil {
		m.Fatal("must read dir", err)
	}
	return ee
}

func (m must) Readlink(name string) string {
	m.Helper()
	item, err := os.Lstat(name)
	if err != nil {
		m.Fatal("must read link", err)
	}

	gotType := item.Mode().Type()
	if gotType != fs.ModeSymlink {
		m.Fatalf("must read link: %q want symlink, got %s", name, gotType)
	}

	gotDest, err := os.Readlink(name)
	if err != nil {
		m.Fatal("must read link", err)
	}
	return gotDest
}

func (m must) Symlink(oldname, newname string) {
	m.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		m.Fatal("must symlink", err)
	}
}

func (m must) WriteFile(path string, data []byte, perm fs.FileMode) {
	m.Helper()
	if err := os.WriteFile(path, data, perm); err != nil {
		m.Fatal("must write file", err)
	}
}
