package duffel

import (
	"errors"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/duftest"
)

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
	const (
		source         = "home/user/source"
		target         = "home/user/target"
		targetToSource = "../source" // Relative path from target to source
	)

	pkgItems := map[string][]struct {
		name  string
		entry *fstest.MapFile
	}{
		"pkg1": {
			{
				name:  "dirItem1",
				entry: duftest.DirEntry(0o755),
			},
			{
				name:  "fileItem1",
				entry: duftest.FileEntry("fileItem1 content", 0o644),
			},
			{
				name:  "linkItem1",
				entry: duftest.LinkEntry("linkItem1/dest"),
			},
		},
		"pkg2": {
			{
				name:  "dirItem2",
				entry: duftest.DirEntry(0o755),
			},
			{
				name:  "fileItem2",
				entry: duftest.FileEntry("fileItem2 content", 0o644),
			},
			{
				name:  "linkItem2",
				entry: duftest.LinkEntry("linkItem2/dest"),
			},
		},
	}

	fsys := duftest.NewFS()
	fsys.M[target] = duftest.DirEntry(0o755)
	for pkg, items := range pkgItems {
		sourcePkg := path.Join(source, pkg)
		for _, item := range items {
			sourcePkgItem := path.Join(sourcePkg, item.name)
			fsys.M[sourcePkgItem] = item.entry
		}
	}

	req := &Request{
		FS:     fsys,
		Source: source,
		Target: target,
		Pkgs:   []string{"pkg1", "pkg2"},
	}

	err := Execute(req, false, nil)
	if err != nil {
		t.Error(err)
	}

	for pkg, items := range pkgItems {
		for _, item := range items {
			wantTargetItem := path.Join(target, item.name)
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

			wantDest := path.Join(targetToSource, pkg, item.name)
			gotDest := string(gotFile.Data)
			if gotDest != wantDest {
				t.Errorf("%q want dest %s, got %s", wantTargetItem, wantDest, gotDest)
			}
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

	files := duftest.NewFS()
	files.M[target] = duftest.DirEntry(0o755)

	// Conflict: pkg2/dirItem and pkg1/dirItem install to same target path
	files.M[path.Join(source, "pkg1/dirItem")] = duftest.DirEntry(0o755)
	files.M[path.Join(source, "pkg2/dirItem")] = duftest.DirEntry(0o755)

	req := &Request{
		FS:     files,
		Source: source,
		Target: target,
		Pkgs:   []string{"pkg1", "pkg2"},
	}

	err := Execute(req, false, nil)

	wantErr := &ErrConflict{}
	if !errors.Is(err, wantErr) {
		t.Errorf("want error %#v, got %#v", wantErr, err)
	}

	if t.Failed() {
		t.Log("files:", files.M)
	}
}

func TestExecuteDirErrors(t *testing.T) {
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
			files := duftest.NewFS()
			wd := "home/user"
			pkg := "pkg"
			absSource := path.Join(wd, "source")
			absSourcePkg := path.Join(absSource, pkg)
			absTarget := path.Join(wd, "target")
			if test.packagePerm != doesNotExist {
				sourcePkgItem := path.Join(absSourcePkg, "item")
				files.M[absSourcePkg] = duftest.DirEntry(test.packagePerm)
				files.M[sourcePkgItem] = duftest.DirEntry(permNormal)
			}
			if test.sourcePerm != doesNotExist {
				files.M[absSource] = duftest.DirEntry(test.sourcePerm)
			}
			if test.targetPerm != doesNotExist {
				files.M[absTarget] = duftest.DirEntry(test.targetPerm)
			}

			r := &Request{
				FS:     files,
				Source: absSource,
				Target: absTarget,
				Pkgs:   []string{pkg},
			}

			err := Execute(r, false, nil)

			if !errors.Is(err, test.wantError) {
				t.Errorf("want error %v, got %v", test.wantError, err)
			}
		})
	}
}
