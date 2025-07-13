package exec

import (
	"io/fs"
	"path"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

type fileDesc struct {
	name string
	mode fs.FileMode
	dest string
}

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
	const (
		source = "home/user/source"
		target = "home/user/target"
	)

	specs := []struct {
		sourceFile fileDesc // Describes a file in the source tree
		targetFile fileDesc // Describes a desired file in the target tree
	}{
		{
			sourceFile: fileDesc{
				name: "pkg1/fileItem1",
				mode: 0o644,
			},
			targetFile: fileDesc{
				name: "fileItem1",
				mode: fs.ModeSymlink,
				dest: "../source/pkg1/fileItem1",
			},
		},
		{
			sourceFile: fileDesc{
				name: "pkg1/dirItem1",
				mode: fs.ModeDir | 0o755,
			},
			targetFile: fileDesc{
				name: "dirItem1",
				mode: fs.ModeSymlink,
				dest: "../source/pkg1/dirItem1",
			},
		},
		{
			sourceFile: fileDesc{
				name: "pkg1/linkItem1",
				mode: fs.ModeSymlink,
				dest: "linkItem1/dest",
			},
			targetFile: fileDesc{
				name: "linkItem1",
				mode: fs.ModeSymlink,
				dest: "../source/pkg1/linkItem1",
			},
		},
		{
			sourceFile: fileDesc{
				name: "pkg2/fileItem2",
				mode: 0o644,
			},
			targetFile: fileDesc{
				name: "fileItem2",
				mode: fs.ModeSymlink,
				dest: "../source/pkg2/fileItem2",
			},
		},
		{
			sourceFile: fileDesc{
				name: "pkg2/dirItem2",
				mode: fs.ModeDir | 0o755,
			},
			targetFile: fileDesc{
				name: "dirItem2",
				mode: fs.ModeSymlink,
				dest: "../source/pkg2/dirItem2",
			},
		},
		{
			sourceFile: fileDesc{
				name: "pkg2/linkItem2",
				mode: fs.ModeSymlink,
				dest: "linkItem2/dest",
			},
			targetFile: fileDesc{
				name: "linkItem2",
				mode: fs.ModeSymlink,
				dest: "../source/pkg2/linkItem2",
			},
		},
	}

	testFS := errfs.New()
	testFS.AddDir(target, 0o755)
	for _, spec := range specs {
		sf := spec.sourceFile
		sourcePkgItem := path.Join(source, sf.name)
		testFS.AddItem(sourcePkgItem, sf.mode, sf.dest)
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

	for _, spec := range specs {
		want := spec.targetFile
		wantFilePath := path.Join(target, want.name)

		gotFile, err := testFS.Find(wantFilePath)
		if err != nil {
			t.Error(err)
			continue
		}

		gotMode := gotFile.Info.Mode()
		if gotMode != want.mode {
			t.Errorf("%q mode:\n got: %s\nwant: %s",
				wantFilePath, gotMode, want.mode)
		}

		gotDest := gotFile.Dest
		if gotDest != want.dest {
			t.Errorf("%q dest:\n got: %s\nwant: %s",
				wantFilePath, gotDest, want.dest)
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
