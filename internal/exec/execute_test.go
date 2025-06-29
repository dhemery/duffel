package exec

import (
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
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
				entry: &fstest.MapFile{Mode: fs.ModeDir | 0o755},
			},
			{
				name:  "fileItem1",
				entry: &fstest.MapFile{Mode: 0o644},
			},
			{
				name:  "linkItem1",
				entry: &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("linkItem1/dest")},
			},
		},
		"pkg2": {
			{
				name:  "dirItem2",
				entry: &fstest.MapFile{Mode: fs.ModeDir | 0o755},
			},
			{
				name:  "fileItem2",
				entry: &fstest.MapFile{Mode: 0o644},
			},
			{
				name:  "linkItem2",
				entry: &fstest.MapFile{Mode: fs.ModeSymlink, Data: []byte("linkItem2/dest")},
			},
		},
	}

	fsys := duftest.NewFS()
	fsys.M[target] = &fstest.MapFile{Mode: fs.ModeDir | 0o755}
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
				t.Errorf("%q mode: got%s, want %s", wantTargetItem, gotMode, wantMode)
			}

			wantDest := path.Join(targetToSource, pkg, item.name)
			gotDest := string(gotFile.Data)
			if gotDest != wantDest {
				t.Errorf("%q dest: got %q, want %q", wantTargetItem, gotDest, wantDest)
			}
		}
	}

	if t.Failed() {
		t.Error("files:")
		for _, name := range slices.Sorted(maps.Keys(fsys.M)) {
			file := fsys.M[name]
			t.Errorf("   %q: %s %s", name, file.Mode, file.Data)
		}
	}
}

func TestExecuteDirErrors(t *testing.T) {
	const (
		dirNormal       = fs.ModeDir | 0o755
		dirUnreadable   = fs.ModeDir | 0o311
		dirUnsearchable = fs.ModeDir | 0o644
		dirUnwriteable  = fs.ModeDir | 0o555
	)

	tests := map[string]struct {
		sourceEntry  *fstest.MapFile
		targetEntry  *fstest.MapFile
		packageEntry *fstest.MapFile
		wantError    error
	}{
		"package dir missing": {
			sourceEntry:  &fstest.MapFile{Mode: dirNormal},
			packageEntry: nil,
			targetEntry:  &fstest.MapFile{Mode: dirNormal},
			wantError:    fs.ErrNotExist,
		},
		"package dir not readable": {
			sourceEntry:  &fstest.MapFile{Mode: dirNormal},
			packageEntry: &fstest.MapFile{Mode: dirUnreadable},
			targetEntry:  &fstest.MapFile{Mode: dirNormal},
			wantError:    fs.ErrPermission,
		},
		"source dir missing": {
			sourceEntry:  nil,
			packageEntry: nil, // Creating package would require source to exist
			targetEntry:  &fstest.MapFile{Mode: dirNormal},
			wantError:    fs.ErrNotExist,
		},
		"source dir not searchable": {
			sourceEntry:  &fstest.MapFile{Mode: dirUnsearchable},
			packageEntry: &fstest.MapFile{Mode: dirNormal},
			targetEntry:  &fstest.MapFile{Mode: dirNormal},
			wantError:    fs.ErrPermission,
		},
		"target dir missing": {
			sourceEntry:  &fstest.MapFile{Mode: dirNormal},
			packageEntry: &fstest.MapFile{Mode: dirNormal},
			targetEntry:  nil,
			wantError:    fs.ErrNotExist,
		},
		"target dir not writeable": {
			sourceEntry:  &fstest.MapFile{Mode: dirNormal},
			packageEntry: &fstest.MapFile{Mode: dirNormal},
			targetEntry:  &fstest.MapFile{Mode: dirUnwriteable},
			wantError:    fs.ErrPermission,
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
			if test.packageEntry != nil {
				sourcePkgItem := path.Join(absSourcePkg, "item")
				files.M[absSourcePkg] = test.packageEntry
				files.M[sourcePkgItem] = &fstest.MapFile{Mode: 0o644} // plain file
			}
			if test.sourceEntry != nil {
				files.M[absSource] = test.sourceEntry
			}
			if test.targetEntry != nil {
				files.M[absTarget] = test.targetEntry
			}

			r := &Request{
				FS:     files,
				Source: absSource,
				Target: absTarget,
				Pkgs:   []string{pkg},
			}

			err := Execute(r, false, nil)

			if !errors.Is(err, test.wantError) {
				t.Errorf("Execute(): got %v, want %v", err, test.wantError)
			}
		})
	}
}
