package duffel

import (
	"bytes"
	"errors"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/testfs"
)

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
	const (
		source = "home/user/source"
		target = "home/user/target"
		// Want each installed link to start with the relative path from target to source
		wantLinkPrefix = "../source"
	)
	items := []struct {
		pkgItem string          // path from source to package item
		file    *fstest.MapFile // item file
		item    string          // desired path from target to installed item
	}{
		// pkg1 items
		{
			pkgItem: "pkg1/dirItem1",
			item:    "dirItem1",
			file:    testfs.DirEntry(0o755),
		},
		{
			pkgItem: "pkg1/fileItem1",
			item:    "fileItem1",
			file:    testfs.FileEntry("fileItem1 content", 0o644),
		},
		{
			pkgItem: "pkg1/linkItem1",
			item:    "linkItem1",
			file:    testfs.LinkEntry("linkItem1/dest"),
		},
		// pkg2 items
		{
			pkgItem: "pkg2/dirItem2",
			item:    "dirItem2",
			file:    testfs.DirEntry(0o755),
		},
		{
			pkgItem: "pkg2/fileItem2",
			item:    "fileItem2",
			file:    testfs.FileEntry("fileItem2 content", 0o644),
		},
		{
			pkgItem: "pkg2/linkItem2",
			item:    "linkItem2",
			file:    testfs.LinkEntry("linkItem2/dest"),
		},
	}

	fsys := testfs.New()
	fsys.M[target] = testfs.DirEntry(0o755)
	for _, item := range items {
		source := path.Join(source, item.pkgItem)
		fsys.M[source] = item.file
	}

	req := &Request{
		FS:     fsys,
		Stdout: &bytes.Buffer{},
		Source: source,
		Target: target,
		Pkgs:   []string{"pkg1", "pkg2"},
	}

	err := Execute(req)
	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		wantTargetItem := path.Join(target, item.item)
		gotFile, ok := fsys.M[wantTargetItem]
		if !ok {
			t.Error("not installed:", wantTargetItem)
			continue
		}

		wantMode := fs.ModeSymlink
		gotMode := gotFile.Mode
		if gotMode != wantMode {
			t.Errorf("%q want mode %s, got %s", wantTargetItem, wantMode, gotMode)
		}

		wantDest := path.Join(wantLinkPrefix, item.pkgItem)
		gotDest := string(gotFile.Data)
		if gotDest != wantDest {
			t.Errorf("%q want dest %s, got %s", wantTargetItem, wantDest, gotDest)
		}
	}

	if t.Failed() {
		t.Log("files:", fsys.M)
	}
}

func TestExecuteEmptyTargetWithConflictingPackageItems(t *testing.T) {
	const (
		source = "home/user/source"
		target = "home/user/target"
	)

	files := testfs.New()
	files.M[target] = testfs.DirEntry(0o755)

	// Conflict: pkg2/dirItem and pkg1/dirItem install to same target path
	files.M[path.Join(source, "pkg1/dirItem")] = testfs.DirEntry(0o755)
	files.M[path.Join(source, "pkg2/dirItem")] = testfs.DirEntry(0o755)

	req := &Request{
		FS:     files,
		Stdout: &bytes.Buffer{},
		Source: source,
		Target: target,
		Pkgs:   []string{"pkg1", "pkg2"},
	}

	wantErr := &Conflict{}
	gotErr := Execute(req)
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("want error %#v, got %#v", wantErr, gotErr)
	}

	if t.Failed() {
		t.Log("files:", files.M)
	}
}

func TestInstallDirErrors(t *testing.T) {
	const (
		doesNotExist = 0
		permRead     = 0o444
		permWrite    = 0o222
		permSearch   = 0o111
		permNormal   = permRead | permWrite | permSearch
		permNoRead   = permNormal ^ permRead
		permNoWrite  = permNormal ^ permWrite
		permNoSearch = permNormal ^ permSearch
	)

	tests := map[string]struct {
		sourcePerm  fs.FileMode
		targetPerm  fs.FileMode
		packagePerm fs.FileMode
		wantError   error
	}{
		"package dir missing": {
			sourcePerm:  permNormal,
			packagePerm: doesNotExist,
			targetPerm:  permNormal,
			wantError:   fs.ErrNotExist,
		},
		"package dir not readable": {
			sourcePerm:  permNormal,
			packagePerm: permNoRead,
			targetPerm:  permNormal,
			wantError:   fs.ErrPermission,
		},
		"source dir missing": {
			sourcePerm:  doesNotExist,
			packagePerm: doesNotExist, // Creating package would require source to exist
			targetPerm:  permNormal,
			wantError:   fs.ErrNotExist,
		},
		"source dir not searchable": {
			sourcePerm:  permNoSearch,
			packagePerm: permNormal,
			targetPerm:  permNormal,
			wantError:   fs.ErrPermission,
		},
		"target dir missing": {
			sourcePerm:  permNormal,
			packagePerm: permNormal,
			targetPerm:  doesNotExist,
			wantError:   fs.ErrNotExist,
		},
		"target dir not writeable": {
			sourcePerm:  permNormal,
			packagePerm: permNormal,
			targetPerm:  permNoWrite,
			wantError:   fs.ErrPermission,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			files := testfs.New()
			wd := "home/user"
			packageName := "pkg"
			absSource := path.Join(wd, "source")
			absSourcePkg := path.Join(absSource, packageName)
			absTarget := path.Join(wd, "target")
			if test.packagePerm != doesNotExist {
				sourcePkgItem := path.Join(absSourcePkg, "item")
				files.M[absSourcePkg] = testfs.DirEntry(test.packagePerm)
				files.M[sourcePkgItem] = testfs.DirEntry(permNormal)
			}
			if test.sourcePerm != doesNotExist {
				files.M[absSource] = testfs.DirEntry(test.sourcePerm)
			}
			if test.targetPerm != doesNotExist {
				files.M[absTarget] = testfs.DirEntry(test.targetPerm)
			}

			r := &Request{
				Stdout: &bytes.Buffer{},
				FS:     files,
				Source: absSource,
				Target: absTarget,
				Pkgs:   []string{packageName},
			}

			gotError := Execute(r)
			if !errors.Is(gotError, test.wantError) {
				t.Errorf("want error %v, got %v", test.wantError, gotError)
			}
		})
	}
}
