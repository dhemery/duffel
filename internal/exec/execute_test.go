package exec

import (
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/google/go-cmp/cmp"
)

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
	const (
		source = "home/user/source"
		target = "home/user/target"
	)

	pkgItems := map[string][]*errfs.File{
		"pkg1": {
			errfs.NewDir("dirItem1", 0o755),
			errfs.NewFile("fileItem1", 0o644),
			errfs.NewSymlink("linkItem1", "linkItem1/dest"),
		},
		"pkg2": {
			errfs.NewDir("dirItem2", 0o755),
			errfs.NewFile("fileItem2", 0o644),
			errfs.NewSymlink("linkItem2", "linkItem2/dest"),
		},
	}

	testFS := errfs.New()
	testFS.Add(path.Dir(target), errfs.NewDir(path.Base(target), 0o755))
	for pkg, itemFiles := range pkgItems {
		sourcePkg := path.Join(source, pkg)
		for _, file := range itemFiles {
			testFS.Add(sourcePkg, file)
		}
	}

	req := &Request{
		FS:     testFS,
		Source: source,
		Target: target,
		Pkgs:   []string{"pkg1", "pkg2"},
	}

	err := Execute(req, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	for pkg, itemFiles := range pkgItems {
		for _, itemFile := range itemFiles {
			wantTargetItem := path.Join(target, itemFile.Name())

			gotFile, err := testFS.Find(wantTargetItem)
			if err != nil {
				t.Error(err)
				continue
			}

			sourcePkgItem := path.Join(source, pkg, itemFile.Name())
			wantLinkDest, _ := filepath.Rel(path.Dir(wantTargetItem), sourcePkgItem)
			wantFile := errfs.NewSymlink(itemFile.Name(), wantLinkDest)

			if !cmp.Equal(gotFile, wantFile) {
				t.Errorf("target file %q:\n got: %s\nwant: %s",
					wantTargetItem, gotFile, wantFile)
			}
		}
	}

	if t.Failed() {
		printFiles(t, testFS, "files after failure:")
	}
}

func printFiles(t *testing.T, fsys fs.FS, context string) {
	t.Helper()
	t.Error(context)
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		t.Errorf("   %q: %v", name, entry)
		return nil
	})
	if err != nil {
		t.Errorf("walk error: %s", err)
	}
}
