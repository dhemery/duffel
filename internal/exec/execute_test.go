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
		targetFile fileDesc // Describes a desired symlink in the target tree
	}{
		{
			sourceFile: fileDesc{
				name: "pkg1/fileItem1",
				mode: 0o644,
			},
			targetFile: fileDesc{
				name: "fileItem1",
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
				dest: "../source/pkg2/linkItem2",
			},
		},
	}

	testFS := errfs.New()
	testFS.AddDir(target, 0o755)
	for _, spec := range specs {
		sf := spec.sourceFile
		sourcePkgItem := path.Join(source, sf.name)
		testFS.Add(sourcePkgItem, sf.mode, sf.dest)
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

		gotInfo, err := testFS.Lstat(wantFilePath)
		if err != nil {
			t.Error(err)
			continue
		}

		gotMode := gotInfo.Mode()
		if gotMode != fs.ModeSymlink {
			t.Errorf("%q mode:\n got: %s\nwant: %s",
				wantFilePath, gotMode, fs.ModeSymlink)
		}

		gotDest, err := testFS.ReadLink(wantFilePath)
		if err != nil {
			t.Error(err)
			continue
		}

		if gotDest != want.dest {
			t.Errorf("%q dest:\n got: %s\nwant: %s",
				wantFilePath, gotDest, want.dest)
		}

	}

	if t.Failed() {
		t.Log("files after failure:")
		t.Error(testFS.String())
	}
}
