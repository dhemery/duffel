package file_test

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/file"
)

func TestRootFSLstat(t *testing.T) {
	must := duftest.Must(t)
	tdir := t.TempDir()
	must.MkdirAll(filepath.Join(tdir, "sub/dir"), 0o755)
	must.WriteFile(filepath.Join(tdir, "sub/file"), []byte{}, 0o644)
	must.Symlink("../ignored/dest", filepath.Join(tdir, "sub/link"))

	fsys := file.RootFS(must.OpenRoot(tdir))

	info, err := fsys.Lstat("sub/dir")

	if err != nil {
		t.Errorf("Lstat(%s): unexpected error: %s", "sub/dir", err)
	} else if !info.IsDir() {
		t.Errorf("Lstat(%s) got %s, want dir", "sub/dir", fs.FormatFileInfo(info))
	}

	info, err = fsys.Lstat("sub/file")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if !info.Mode().IsRegular() {
		t.Errorf("Lstat(%s) got %s, want regular file", "sub/file", fs.FormatFileInfo(info))
	}

	info, err = fsys.Lstat("sub/link")
	if err != nil {
		t.Error("unexpected error:", err)
	} else if info.Mode()&fs.ModeType != fs.ModeSymlink {
		t.Errorf("Lstat(%s) got %s, want symlink", "sub/link", fs.FormatFileInfo(info))
	}

	info, err = fsys.Lstat("no/such/entry")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Lstat(%s) error: got %s, want error %s", "no/such/entry", err, fs.ErrNotExist)
	}
	if info != nil {
		t.Errorf("Lstat(%s) info: got %s, want nil", "no/such/entry", fs.FormatFileInfo(info))
	}
}

func TestRootFSMkdir(t *testing.T) {
	must := duftest.Must(t)
	tdir := t.TempDir()

	existingPerm := fs.FileMode(0o755)
	existingDir := "existing-dir"
	must.MkdirAll(filepath.Join(tdir, existingDir), existingPerm)

	newPerm := fs.FileMode(0o700)
	newDir := "existing-dir/new-dir"

	fsys := file.RootFS(must.OpenRoot(tdir))

	err := fsys.Mkdir(newDir, newPerm)
	if err != nil {
		t.Fatalf("MkDir(%s): unexpected error: %s", "existing-dir/new-dir", err)
	}

	info := must.Lstat(filepath.Join(tdir, newDir))
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

func TestRootFSReadDir(t *testing.T) {
	must := duftest.Must(t)
	tdir := t.TempDir()
	dirPerm := fs.FileMode(0o755)
	must.MkdirAll(filepath.Join(tdir, "sub/dir"), dirPerm)
	filePerm := fs.FileMode(0o644)
	must.WriteFile(filepath.Join(tdir, "sub/file"), []byte{}, filePerm)
	must.Symlink("../ignored/dest", filepath.Join(tdir, "sub/link"))

	fsys := file.RootFS(must.OpenRoot(tdir))

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
		checkEntryMode(t, e, fs.ModeDir|dirPerm)
	} else {
		t.Error("no entry for", "dir")
	}

	if e, ok := entryNamed["file"]; ok {
		if !e.Type().IsRegular() {
			t.Errorf("%q mode: got %s, want regular file", "file", fs.FormatDirEntry(e))
		}
		checkEntryMode(t, e, filePerm) // No other mode bits on for regular files
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

func TestRootFSRemove(t *testing.T) {
	tests := map[string]struct {
		setup   filesFunc
		remove  string
		wantErr bool
		check   filesFunc
	}{
		"file": {
			setup:   createFile("existing/file", 0o644),
			remove:  "existing/file",
			wantErr: false,
			check:   checkNotExist("existing/file"),
		},
		"empty dir": {
			setup:   createDir("existing/empty/dir", 0o755),
			remove:  "existing/empty/dir",
			wantErr: false,
			check:   checkNotExist("existing/empty/dir"),
		},
		"link": {
			setup:   createLink("some/dest", "existing/link"),
			remove:  "existing/link",
			wantErr: false,
			check:   checkNotExist("existing/link"),
		},
		"non-empty dir": {
			setup:   createFile("existing/dir/not/empty", 0o755),
			remove:  "existing/dir",
			wantErr: true,
			check:   checkMode("existing/dir", fs.ModeDir|0o755),
		},
		"non-existent entry": {
			setup:   nil,
			remove:  "non-existent/entry",
			wantErr: true,
			check:   checkNotExist("non-existent/entry"),
		},
		"invalid path": {
			setup:   nil,
			remove:  "invalid/../path",
			wantErr: true,
			check:   checkNotExist("invalid/../path"),
		},
	}

	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			must := duftest.Must(t)
			tdir := t.TempDir()

			if test.setup != nil {
				test.setup(t, tdir)
			}

			fsys := file.RootFS(must.OpenRoot(tdir))

			err := fsys.Remove(test.remove)
			gotErr := err != nil
			switch {
			case gotErr == test.wantErr:
			case err != nil:
				t.Errorf("Remove(%q) unexpected error: %v", test.remove, err)
			default:
				t.Errorf("Remove(%q) want error, got none", test.remove)
			}

			test.check(t, tdir)
		})
	}
}

func TestRootFSSymlink(t *testing.T) {
	must := duftest.Must(t)
	tdir := t.TempDir()
	fsys := file.RootFS(must.OpenRoot(tdir))

	must.MkdirAll(filepath.Join(tdir, "sub"), 0o755)
	linkDest := "../../some/link/dest"
	goodPath := "sub/link"

	err := fsys.Symlink(linkDest, goodPath)

	if err == nil {
		e := must.Lstat(filepath.Join(tdir, goodPath))
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

func checkEntryMode(t *testing.T, entry fs.DirEntry, wantMode fs.FileMode) {
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

type filesFunc func(t *testing.T, root string)

func createDir(name string, perm fs.FileMode) filesFunc {
	return func(t *testing.T, root string) {
		t.Helper()
		fullname := path.Join(root, name)
		must := duftest.Must(t)
		must.MkdirAll(fullname, perm)
	}
}

func createFile(name string, perm fs.FileMode) filesFunc {
	return func(t *testing.T, root string) {
		t.Helper()
		fullname := path.Join(root, name)
		must := duftest.Must(t)
		must.MkdirAll(path.Dir(fullname), 0o755)
		must.WriteFile(fullname, []byte{}, perm)
	}
}

func createLink(oldname, newname string) filesFunc {
	return func(t *testing.T, root string) {
		t.Helper()
		fullname := path.Join(root, newname)
		must := duftest.Must(t)
		must.MkdirAll(path.Dir(fullname), 0o755)
		must.Symlink(oldname, fullname)
	}
}

func checkMode(name string, wantMode fs.FileMode) filesFunc {
	return func(t *testing.T, root string) {
		t.Helper()
		fullname := path.Join(root, name)
		info, err := os.Lstat(fullname)
		if err != nil {
			t.Errorf("checkMode(%q):\n got error: %v\nwant mode: %s",
				name, errors.Unwrap(err), wantMode.String())
			return
		}
		gotMode := info.Mode()
		if gotMode != wantMode {
			t.Errorf("checkMode(%q) mode:\n got: %s\nwant: %s",
				name, gotMode.String(), wantMode.String())
		}
	}
}

func checkNotExist(name string) filesFunc {
	return func(t *testing.T, root string) {
		t.Helper()
		fullname := path.Join(root, name)
		info, err := os.Lstat(fullname)
		if err == nil {
			t.Errorf("checkNotExist(%q) got %s",
				name, info.Mode().String())
		}
	}
}
