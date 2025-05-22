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
	e := mustLstat(t, filepath.Join(root, newParent))
	if !e.IsDir() {
		t.Errorf("%q want dir, got %s", newParent, fs.FormatFileInfo(e))
	}
	if e.Mode().Perm() != newPerm {
		t.Errorf("%q want permission %O, got %s", newParent, newPerm, fs.FormatFileInfo(e))
	}

	e = mustLstat(t, filepath.Join(root, newDir))
	if !e.IsDir() {
		t.Errorf("%q want dir, got %s", newDir, fs.FormatFileInfo(e))
	}
	if e.Mode().Perm() != newPerm {
		t.Errorf("%q want permission, got %s", newDir, fs.FormatFileInfo(e))
	}
}

func TestDirFSReadDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	dirPerm := fs.FileMode(0o755)
	mustMkdirAll(t, filepath.Join(root, "sub/dir"), dirPerm)
	filePerm := fs.FileMode(0o644)
	mustWriteFile(t, filepath.Join(root, "sub/file"), []byte{}, filePerm)
	mustSymlink(t, "../ignored/dest", filepath.Join(root, "sub/link"))

	f := dirFS(root)

	entries, err := f.ReadDir("sub")
	if err != nil {
		t.Error("unexpected error:", err)
	}
	entryNamed := map[string]fs.DirEntry{}
	for _, e := range entries {
		entryNamed[e.Name()] = e
	}

	if e, ok := entryNamed["dir"]; ok {
		if !e.IsDir() {
			t.Errorf("%q want dir", fs.FormatDirEntry(e))
		}
		assertEntryMode(t, e, fs.ModeDir|dirPerm)
	} else {
		t.Error("no entry for", "dir")
	}

	if e, ok := entryNamed["file"]; ok {
		if !e.Type().IsRegular() {
			t.Errorf("%q want regular file, got %s", "file", fs.FormatDirEntry(e))
		}
		assertEntryMode(t, e, filePerm) // No other mode bits on for regular files
	} else {
		t.Error("no entry for", "file")
	}

	if e, ok := entryNamed["link"]; ok {
		if e.Type()&fs.ModeType != fs.ModeSymlink {
			t.Errorf("%q want symlink", fs.FormatDirEntry(e))
		}
	} else {
		t.Error("no entry for", "link")
	}
}

func TestDirFSSymlink(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	mustMkdirAll(t, filepath.Join(root, "sub"), 0o755)

	f := dirFS(root)
	linkDest := "../../some/link/dest"

	goodPath := "sub/link"
	err := f.Symlink(linkDest, goodPath)

	if err == nil {
		e := mustLstat(t, filepath.Join(root, goodPath))
		gotType := e.Mode().Type()
		if gotType != fs.ModeSymlink {
			t.Errorf("%q want symlink, got %s", e.Name(), &gotType)
		}
	} else {
		t.Error("unexpected error:", err)
	}

	badPath := "nonexistent-parent/link"
	err = f.Symlink(linkDest, badPath)
	wantErr := fs.ErrNotExist
	if !errors.Is(err, wantErr) {
		t.Errorf("%q want error %q, got %q", badPath, wantErr, err)
	}
}

func assertEntryMode(t *testing.T, entry fs.DirEntry, wantMode fs.FileMode) {
	t.Helper()
	info, err := entry.Info()
	if err != nil {
		t.Errorf("%q.Info(): %s", entry, err)
		return
	}
	gotMode := info.Mode()
	if gotMode != wantMode {
		t.Errorf("%q want mode %O, got %O",
			fs.FormatDirEntry(entry), wantMode, gotMode)
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
	item, err := os.Lstat(path)
	if err != nil {
		t.Fatal("must read link", err)
	}

	gotType := item.Mode().Type()
	if gotType != fs.ModeSymlink {
		t.Fatalf("%q want symlink, got %s", path, gotType)
	}

	gotDest, err := os.Readlink(path)
	if err != nil {
		t.Fatal("must read link", err)
	}
	return gotDest
}

func mustLstat(t *testing.T, path string) fs.FileInfo {
	t.Helper()
	e, err := os.Lstat(path)
	if err != nil {
		t.Fatal("must lstat", err)
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
