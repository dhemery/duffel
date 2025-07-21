package exec

import (
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestExecuteEmptyTargetNoConflictingPackageItems(t *testing.T) {
	specs := []struct {
		sourceFile *errfs.ErrFile // A file in the source tree.
		targetFile *errfs.ErrFile // A desired symlink in the target tree.
	}{
		{
			sourceFile: errfs.File("source/pkg1/fileItem1", 0o644),
			targetFile: errfs.Link("target/fileItem1", "../source/pkg1/fileItem1"),
		},
		{
			sourceFile: errfs.Dir("source/pkg1/dirItem1", 0o755),
			targetFile: errfs.Link("target/dirItem1", "../source/pkg1/dirItem1"),
		},
		{
			sourceFile: errfs.Link("source/pkg1/linkItem1", "linkItem1/dest"),
			targetFile: errfs.Link("target/linkItem1", "../source/pkg1/linkItem1"),
		},
		{
			sourceFile: errfs.File("source/pkg2/fileItem2", 0o644),
			targetFile: errfs.Link("target/fileItem2", "../source/pkg2/fileItem2"),
		},
		{
			sourceFile: errfs.Dir("source/pkg2/dirItem2", 0o755),
			targetFile: errfs.Link("target/dirItem2", "../source/pkg2/dirItem2"),
		},
		{
			sourceFile: errfs.Link("source/pkg2/linkItem2", "linkItem2/dest"),
			targetFile: errfs.Link("target/linkItem2", "../source/pkg2/linkItem2"),
		},
	}

	testFS := errfs.New()
	testFS.Add(errfs.Dir("target", 0o755))
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
		gotFile, err := testFS.Find(wantFile.FullName())
		if err != nil {
			t.Error(err)
			continue
		}

		if !gotFile.Equal(wantFile) {
			t.Errorf("%q:\n got: %s\nwant: %s",
				wantFile.FullName(), gotFile, wantFile)
		}

	}

	if t.Failed() {
		t.Log("files after failure:")
		t.Error(testFS.String())
	}
}
