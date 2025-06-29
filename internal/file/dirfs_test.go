package file

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

	info, err := fs.Lstat(fsys, "sub/dir")
	if err != nil {
		t.Errorf("Lstat(%s): unexpected error: %s", "sub/dir", err)
	} else if !info.IsDir() {
		t.Errorf("Lstat(%s) got %s, want dir", "sub/dir", fs.FormatFileInfo(info))
	}

	info, err = fs.Lstat(fsys, "sub/file")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if !info.Mode().IsRegular() {
		t.Errorf("Lstat(%s) got %s, want regular file", "sub/file", fs.FormatFileInfo(info))
	}

	info, err = fs.Lstat(fsys, "sub/link")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if info.Mode()&fs.ModeType != fs.ModeSymlink {
		t.Errorf("Lstat(%s) got %s, want symlink", "sub/link", fs.FormatFileInfo(info))
	}

	info, err = fs.Lstat(fsys, "no/such/entry")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Lstat(%s) error: got %s, want error %s", "no/such/entry", err, fs.ErrNotExist)
	}
	if info != nil {
		t.Errorf("Lstat(%s) info: got %s, want nil", "no/such/entry", fs.FormatFileInfo(info))
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
		t.Fatalf("MkDir(%s): unexpected error: %s", "existing-dir/new-dir", err)
	}

	info := must.Lstat(filepath.Join(root, newDir))
	if !info.IsDir() {
		t.Errorf("Lstat(%s) info: got %s, want dir", newDir, fs.FormatFileInfo(info))
	}
	if info.Mode().Perm() != newPerm {
		t.Errorf("Lstat(%s) perm: got %s, want %s", newDir, fs.FormatFileInfo(info), newPerm)
	}

	badDir := "no-such-parent/new-dir"
	gotErr := fsys.Mkdir(badDir, 0o755)
	wantErr := fs.ErrNotExist
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("Mkdir(%s) error: got %v, want %s", badDir, gotErr, wantErr)
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

	entries, err := fs.ReadDir(fsys, "sub")
	if err != nil {
		t.Errorf("ReadDir(%s) unexpected error: %s", "sub", err)
	}
	entryNamed := map[string]fs.DirEntry{}
	for _, e := range entries {
		entryNamed[e.Name()] = e
	}

	if e, ok := entryNamed["dir"]; ok {
		if !e.IsDir() {
			t.Errorf("%q mode: got %s, want dir", "dir", fs.FormatDirEntry(e))
		}
		assertEntryMode(t, e, fs.ModeDir|dirPerm)
	} else {
		t.Error("no entry for", "dir")
	}

	if e, ok := entryNamed["file"]; ok {
		if !e.Type().IsRegular() {
			t.Errorf("%q mode: got %s, want regular file", "file", fs.FormatDirEntry(e))
		}
		assertEntryMode(t, e, filePerm) // No other mode bits on for regular files
	} else {
		t.Error("no entry for", "file")
	}

	if e, ok := entryNamed["link"]; ok {
		if e.Type()&fs.ModeType != fs.ModeSymlink {
			t.Errorf("%q mode: got %s, want symlink", "link", fs.FormatDirEntry(e))
		}
	} else {
		t.Errorf("ReadDir(%s): no entry for link", "sub")
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
			t.Errorf("Lstat(%s) type: got %s, want symlink", e.Name(), &gotType)
		}
	} else {
		t.Error("unexpected error:", err)
	}

	badPath := "nonexistent-parent/link"
	err = fsys.Symlink(linkDest, badPath)
	wantErr := fs.ErrNotExist
	if !errors.Is(err, wantErr) {
		t.Errorf("Symlink(%s) error: got %s, want %s", badPath, err, wantErr)
	}
}

func TestValidatesPath(t *testing.T) {
	root := t.TempDir()

	fsys := DirFS(root)

	_, err := fs.Lstat(fsys, "foo/../lstat")
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("lstat got %v, want %v", err, fs.ErrInvalid)
	}

	err = fsys.Mkdir("foo/../mkdir", 0o755)
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("mkdir got %v, want %v", err, fs.ErrInvalid)
	}

	err = fsys.Symlink("link/dest", "foo/../symlink")
	if !errors.Is(err, fs.ErrInvalid) {
		t.Errorf("symlink got %v, want %v", err, fs.ErrInvalid)
	}
}

func assertEntryMode(t *testing.T, entry fs.DirEntry, wantMode fs.FileMode) {
	t.Helper()
	info, err := entry.Info()
	if err != nil {
		t.Errorf("Info(%s): %s", entry, err)
		return
	}
	gotMode := info.Mode()
	if gotMode != wantMode {
		t.Errorf("%q mode: got %O, want %O",
			fs.FormatDirEntry(entry), gotMode, wantMode)
	}
}
