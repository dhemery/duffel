// Package filestest implements helpers for tests that interact with the file system.
package filestest

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

func (m must) MkdirAll(dir string, perm fs.FileMode) {
	m.Helper()
	if err := os.MkdirAll(dir, perm); err != nil {
		m.Fatal("must mkdir all", err)
	}
}

func (m must) Readlink(path string) string {
	m.Helper()
	item, err := os.Lstat(path)
	if err != nil {
		m.Fatal("must read link", err)
	}

	gotType := item.Mode().Type()
	if gotType != fs.ModeSymlink {
		m.Fatalf("must read link: %q want symlink, got %s", path, gotType)
	}

	gotDest, err := os.Readlink(path)
	if err != nil {
		m.Fatal("must read link", err)
	}
	return gotDest
}

func (m must) Lstat(path string) fs.FileInfo {
	m.Helper()
	e, err := os.Lstat(path)
	if err != nil {
		m.Fatal("must lstat", err)
	}
	return e
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
