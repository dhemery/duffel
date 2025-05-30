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
		source string          // path from source to package item
		file   *fstest.MapFile // item file
		target string          // desired path from target to installed item
	}{
		// pkg1 items
		{
			source: "pkg1/dirItem1",
			file:   testfs.DirEntry(0o755),
			target: "dirItem1",
		},
		{
			source: "pkg1/fileItem1",
			file:   testfs.FileEntry("fileItem1 content", 0o644),
			target: "fileItem1",
		},
		{
			source: "pkg1/linkItem1",
			file:   testfs.LinkEntry("linkItem1/dest"),
			target: "linkItem1",
		},
		// pkg2 items
		{
			source: "pkg2/dirItem2",
			file:   testfs.DirEntry(0o755),
			target: "dirItem2",
		},
		{
			source: "pkg2/fileItem2",
			file:   testfs.FileEntry("fileItem2 content", 0o644),
			target: "fileItem2",
		},
		{
			source: "pkg2/linkItem2",
			file:   testfs.LinkEntry("linkItem2/dest"),
			target: "linkItem2",
		},
	}

	files := testfs.New()
	files.M[target] = testfs.DirEntry(0o755)
	for _, item := range items {
		source := path.Join(source, item.source)
		files.M[source] = item.file
	}

	req := &Request{
		FS:     files,
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
		wantTargetPath := path.Join(target, item.target)
		gotFile, ok := files.M[wantTargetPath]
		if !ok {
			t.Error("not installed:", wantTargetPath)
			continue
		}

		wantMode := fs.ModeSymlink
		gotMode := gotFile.Mode
		if gotMode != wantMode {
			t.Errorf("%q want mode %s, got %s", wantTargetPath, wantMode, gotMode)
		}

		wantDest := path.Join(wantLinkPrefix, item.source)
		gotDest := string(gotFile.Data)
		if gotDest != wantDest {
			t.Errorf("%q want dest %s, got %s", wantTargetPath, wantDest, gotDest)
		}
	}

	if t.Failed() {
		t.Log("files:", files)
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
			sourcePath := path.Join(wd, "source")
			packagePath := path.Join(sourcePath, packageName)
			targetPath := path.Join(wd, "target")
			if test.packagePerm != doesNotExist {
				itemPath := path.Join(packagePath, "item")
				files.M[packagePath] = testfs.DirEntry(test.packagePerm)
				files.M[itemPath] = testfs.DirEntry(permNormal)
			}
			if test.sourcePerm != doesNotExist {
				files.M[sourcePath] = testfs.DirEntry(test.sourcePerm)
			}
			if test.targetPerm != doesNotExist {
				files.M[targetPath] = testfs.DirEntry(test.targetPerm)
			}

			r := &Request{
				Stdout: &bytes.Buffer{},
				FS:     files,
				Source: sourcePath,
				Target: targetPath,
				Pkgs:   []string{packageName},
			}

			gotError := Execute(r)
			if !errors.Is(gotError, test.wantError) {
				t.Errorf("want error %v, got %v", test.wantError, gotError)
			}
		})
	}
}
