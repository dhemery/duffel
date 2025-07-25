package exec

import (
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/log"
	"github.com/google/go-cmp/cmp"
)

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
	log.Set(log.LevelNone, nil)
	specs := []struct {
		sourceFile *errfs.File // A file in the source tree.
		targetFile *errfs.File // A desired symlink in the target tree.
	}{
		{
			sourceFile: errfs.NewFile("source/pkg1/fileItem1", 0o644),
			targetFile: errfs.NewLink("target/fileItem1", "../source/pkg1/fileItem1"),
		},
		{
			sourceFile: errfs.NewDir("source/pkg1/dirItem1", 0o755),
			targetFile: errfs.NewLink("target/dirItem1", "../source/pkg1/dirItem1"),
		},
		{
			sourceFile: errfs.NewLink("source/pkg1/linkItem1", "linkItem1/dest"),
			targetFile: errfs.NewLink("target/linkItem1", "../source/pkg1/linkItem1"),
		},
		{
			sourceFile: errfs.NewFile("source/pkg2/fileItem2", 0o644),
			targetFile: errfs.NewLink("target/fileItem2", "../source/pkg2/fileItem2"),
		},
		{
			sourceFile: errfs.NewDir("source/pkg2/dirItem2", 0o755),
			targetFile: errfs.NewLink("target/dirItem2", "../source/pkg2/dirItem2"),
		},
		{
			sourceFile: errfs.NewLink("source/pkg2/linkItem2", "linkItem2/dest"),
			targetFile: errfs.NewLink("target/linkItem2", "../source/pkg2/linkItem2"),
		},
	}

	testFS := errfs.New()
	errfs.AddDir(testFS, "target", 0o755)
	for _, spec := range specs {
		errfs.Add(testFS, spec.sourceFile)
	}

	req := &Request{
		FS:     testFS,
		Source: "source",
		Target: "target",
		Pkgs:   []string{"pkg1", "pkg2"},
	}

	err := Execute(req, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, spec := range specs {
		wantFile := spec.targetFile
		wantFileName := errfs.FileName(wantFile)
		gotFile, err := errfs.Find(testFS, wantFileName)
		if err != nil {
			t.Error(err)
			continue
		}

		if diff := cmp.Diff(wantFile, gotFile, compareFiles()); diff != "" {
			t.Errorf("%q:\n%s", wantFileName, diff)
		}
	}

	if t.Failed() {
		t.Log("files after failure:")
		t.Log(testFS.String())
	}
}

func compareFiles() cmp.Option {
	return cmp.AllowUnexported(errfs.File{})
}
