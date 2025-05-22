package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestDirFSLstat(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	mustMkdirAll(t, filepath.Join(root, "sub/dir"), 0o755)
	mustWriteFile(t, filepath.Join(root, "sub/file"), []byte{}, 0o644)
	mustSymlink(t, "../ignored/dest", filepath.Join(root, "sub/link"))

	f := dirFS(root)

	e, err := f.Lstat("sub/dir")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if !e.IsDir() {
		t.Errorf("%q want dir, got %s", "sub/dir", fs.FormatFileInfo(e))
	}

	e, err = f.Lstat("sub/file")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if !e.Mode().IsRegular() {
		t.Errorf("%q want regular file, got %s", "sub/file", fs.FormatFileInfo(e))
	}

	e, err = f.Lstat("sub/link")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if e.Mode()&fs.ModeType != fs.ModeSymlink {
		t.Errorf("%q want symlink got %s", "sub/link", fs.FormatFileInfo(e))
	}

	e, err = f.Lstat("no/such/entry")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%q want error %s, got %s", "no/such/entry", fs.ErrNotExist, err)
	}
	if e != nil {
		t.Errorf("%q want entry to be nil, got %s", "no/such/entry", fs.FormatFileInfo(e))
	}
}

func TestDirFSMkdirAll(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	f := dirFS(root)

	existingPerm := fs.FileMode(0o755)
	existingDir := "existing-dir"
	mustMkdirAll(t, filepath.Join(root, existingDir), existingPerm)

	newPerm := fs.FileMode(0o700)
	newDir := "existing-dir/new-parent/new-dir"
	err := f.MkdirAll(newDir, newPerm)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	newParent := "existing-dir/new-parent"
	e := mustStat(t, filepath.Join(root, newParent))
	if !e.IsDir() {
		t.Errorf("%q want dir, got %s", newParent, fs.FormatFileInfo(e))
	}
	if e.Mode().Perm() != newPerm {
		t.Errorf("%q want permission %O, got %s", newParent, newPerm, fs.FormatFileInfo(e))
	}

	e = mustStat(t, filepath.Join(root, newDir))
	if !e.IsDir() {
		t.Errorf("%q want dir, got %s", newDir, fs.FormatFileInfo(e))
	}
	if e.Mode().Perm() != newPerm {
		t.Errorf("%q want permission, got %s", newDir, fs.FormatFileInfo(e))
	}
}

func mustMkdirAll(t *testing.T, dir string, perm fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(dir, perm); err != nil {
		t.Fatal("must mkdir all", err)
	}
}

func mustReadlink(t *testing.T, path string) string {
	t.Helper()
	entry, err := os.Lstat(path)
	if err != nil {
		t.Fatal("must read lin", err)
	}

	gotType := entry.Mode() & fs.ModeType
	if gotType != fs.ModeSymlink {
		t.Fatalf("want a link (file type %O), got file %q is type %O\n", fs.ModeSymlink, path, gotType)
	}

	gotDest, err := os.Readlink(path)
	if err != nil {
		t.Fatal("must read lin", err)
	}
	return gotDest
}

func mustStat(t *testing.T, path string) fs.FileInfo {
	t.Helper()
	e, err := os.Stat(path)
	if err != nil {
		t.Fatal("must stat", err)
	}
	return e
}

func mustSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Fatal("must symlink", err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte, perm fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatal("must write file", err)
	}
}
