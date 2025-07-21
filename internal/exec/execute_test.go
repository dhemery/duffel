package exec

import (
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
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
	testFS.Add(errfs.NewDir("target", 0o755))
	for _, spec := range specs {
		testFS.Add(spec.sourceFile)
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
		gotFile, err := testFS.Find(wantFile.Name)
		if err != nil {
			t.Error(err)
			continue
		}

		fileDiff := cmp.Diff(wantFile, gotFile, cmpopts.IgnoreUnexported(errfs.File{}))
		if fileDiff != "" {
			t.Errorf("%q:\n%s", wantFile.Name, fileDiff)
		}

	}

	if t.Failed() {
		t.Log("files after failure:")
		t.Error(testFS.String())
	}
}
