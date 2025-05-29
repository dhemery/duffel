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

func TestInstallFreshTargetOnePackage(t *testing.T) {
	const (
		pkgName = "pkg"
		source  = "home/user/source"
		target  = "home/user/target"
	)
	items := []struct {
		source   string
		target   string
		wantDest string
		info     *fstest.MapFile
	}{
		{
			source:   "home/user/source/pkg/dirItem",
			target:   "home/user/target/dirItem",
			wantDest: "../source/pkg/dirItem",
			info:     testfs.DirEntry(0o755),
		},
		{
			source:   "home/user/source/pkg/fileItem",
			target:   "home/user/target/fileItem",
			wantDest: "../source/pkg/fileItem",
			info:     testfs.FileEntry("ignored content", 0o644),
		},
		{
			source:   "home/user/source/pkg/linkItem",
			target:   "home/user/target/linkItem",
			wantDest: "../source/pkg/linkItem",
			info:     testfs.LinkEntry("ignored/link/dest"),
		},
	}

	files := testfs.New()
	files.M[target] = testfs.DirEntry(0o755)
	for _, item := range items {
		files.M[item.source] = item.info
	}

	req := &Request{
		FS:     files,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Source: source,
		Target: target,
		Pkgs:   []string{pkgName},
	}

	err := Execute(req)
	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		installed, ok := files.M[item.target]
		if !ok {
			t.Errorf("%q not installed:", item.target)
			continue
		}
		wantMode := fs.ModeSymlink
		if installed.Mode != wantMode {
			t.Errorf("%q want mode %s, got %s", item.target, wantMode, installed.Mode)
		}
		gotDest := string(installed.Data)
		if gotDest != item.wantDest {
			t.Errorf("%q want dest %s, got %s", item.target, item.wantDest, gotDest)
		}
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
	type entry struct {
		path string
		mode fs.FileMode
	}

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
				Stderr: &bytes.Buffer{},
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
