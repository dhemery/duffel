package rules

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

type duffelPathTest struct {
	FS   fs.FS
	Path string
	Want error
}

var duffelPathTests = map[string]duffelPathTest{
	"path to dir with .duffel file is valid": {
		FS: fstest.MapFS{
			"path/to/dir-with-duffel-file/.duffel": regularFile(),
		},
		Path: "path/to/dir-with-duffel-file",
		Want: nil,
	},
	"path to dir with no .duffel file is invalid": {
		FS: fstest.MapFS{
			"path/to/dir-with-no-duffel-file": directory(0755),
		},
		Path: "path/to/dir-with-no-duffel-file",
		Want: ErrNotDuffelDir,
	},
	"path to dir with .duffel dir is invalid": {
		FS: fstest.MapFS{
			"path/to/dir-with-duffel-dir/.duffel": directory(0755),
		},
		Path: "path/to/dir-with-duffel-dir",
		Want: ErrNotFile,
	},
	"path to dir with .duffel link is invalid": {
		FS: fstest.MapFS{
			"path/to/dir-with-duffel-link/.duffel": linkTo("some/path"),
		},
		Path: "path/to/dir-with-duffel-link",
		Want: ErrNotFile,
	},
	"path to file is invalid": {
		FS: fstest.MapFS{
			"path/to/file": regularFile(),
		},
		Path: "path/to/file",
		Want: ErrNotDir,
	},
	"path to link is invalid": {
		FS: fstest.MapFS{
			"path/to/link": linkTo("some/place"),
		},
		Path: "path/to/link",
		Want: ErrNotDir,
	},
	"path to nowhere is invalid": {
		FS: fstest.MapFS{
			"path/to/nowhere": nil,
		},
		Path: "path/to/nowhere",
		Want: ErrNotExist,
	},
}

func TestCheckDuffelDirPath(t *testing.T) {
	for name, test := range duffelPathTests {
		t.Run(name, func(t *testing.T) {
			got := CheckIsDuffelDir(test.FS, test.Path)
			if !errors.Is(got, test.Want) {
				t.Errorf("got %v, want %v", got, test.Want)
			}
		})
	}
}
