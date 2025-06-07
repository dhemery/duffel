package files

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
)

func TestLstat(t *testing.T) {
	must := duftest.Must(t)
	root := t.TempDir()
	must.MkdirAll(filepath.Join(root, "sub/dir"), 0o755)
	must.WriteFile(filepath.Join(root, "sub/file"), []byte{}, 0o644)
	must.Symlink("../ignored/dest", filepath.Join(root, "sub/link"))

	fsys := DirFS(root)

	e, err := fsys.Lstat("sub/dir")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if !e.IsDir() {
		t.Errorf("%q want dir, got %s", "sub/dir", fs.FormatFileInfo(e))
	}

	e, err = fsys.Lstat("sub/file")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if !e.Mode().IsRegular() {
		t.Errorf("%q want regular file, got %s", "sub/file", fs.FormatFileInfo(e))
	}

	e, err = fsys.Lstat("sub/link")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if e.Mode()&fs.ModeType != fs.ModeSymlink {
		t.Errorf("%q want symlink got %s", "sub/link", fs.FormatFileInfo(e))
	}

	e, err = fsys.Lstat("no/such/entry")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%q want error %s, got %s", "no/such/entry", fs.ErrNotExist, err)
	}
	if e != nil {
		t.Errorf("%q want entry to be nil, got %s", "no/such/entry", fs.FormatFileInfo(e))
	}
}

func TestMkdir(t *testing.T) {
	must := duftest.Must(t)
	root := t.TempDir()
	fsys := DirFS(root)

	existingPerm := fs.FileMode(0o755)
	existingDir := "existing-dir"
	must.MkdirAll(filepath.Join(root, existingDir), existingPerm)

	newPerm := fs.FileMode(0o700)
	newDir := "existing-dir/new-dir"
	err := fsys.Mkdir(newDir, newPerm)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}

	e := must.Lstat(filepath.Join(root, newDir))
	if !e.IsDir() {
		t.Errorf("%q want dir, got %s", newDir, fs.FormatFileInfo(e))
	}
	if e.Mode().Perm() != newPerm {
		t.Errorf("%q want permission %s, got %s", newDir, newPerm, fs.FormatFileInfo(e))
	}

	badDir := "no-such-parent/new-dir"
	gotErr := fsys.Mkdir(badDir, 0o755)
	wantErr := fs.ErrNotExist
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("%s want error %s, got %v", badDir, wantErr, gotErr)
	}
}

func TestReadDir(t *testing.T) {
	must := duftest.Must(t)
	root := t.TempDir()
	dirPerm := fs.FileMode(0o755)
	must.MkdirAll(filepath.Join(root, "sub/dir"), dirPerm)
	filePerm := fs.FileMode(0o644)
	must.WriteFile(filepath.Join(root, "sub/file"), []byte{}, filePerm)
	must.Symlink("../ignored/dest", filepath.Join(root, "sub/link"))

	fsys := DirFS(root)

	entries, err := fsys.ReadDir("sub")
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

func TestSymlink(t *testing.T) {
	must := duftest.Must(t)
	root := t.TempDir()
	must.MkdirAll(filepath.Join(root, "sub"), 0o755)

	fsys := DirFS(root)
	linkDest := "../../some/link/dest"

	goodPath := "sub/link"
	err := fsys.Symlink(linkDest, goodPath)

	if err == nil {
		e := must.Lstat(filepath.Join(root, goodPath))
		gotType := e.Mode().Type()
		if gotType != fs.ModeSymlink {
			t.Errorf("%q want symlink, got %s", e.Name(), &gotType)
		}
	} else {
		t.Error("unexpected error:", err)
	}

	badPath := "nonexistent-parent/link"
	err = fsys.Symlink(linkDest, badPath)
	wantErr := fs.ErrNotExist
	if !errors.Is(err, wantErr) {
		t.Errorf("%q want error %q, got %q", badPath, wantErr, err)
	}
}

func TestValidatesPath(t *testing.T) {
	root := t.TempDir()

	fsys := DirFS(root)

	_, err := fsys.Lstat("foo/../lstat")
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("lstat want %v, got %v", fs.ErrInvalid, err)
	}

	err = fsys.Mkdir("foo/../mkdir", 0o755)
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("mkdir want %v, got %v", fs.ErrInvalid, err)
	}

	err = fsys.Symlink("link/dest", "foo/../symlink")
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("symlink want %v, got %v", fs.ErrInvalid, err)
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
