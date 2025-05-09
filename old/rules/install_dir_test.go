package rules

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

type installPathTest struct {
	FS   fs.FS
	Path string
	Want error
}

var installPathTests = map[string]installPathTest{
	"path to readable dir is good": {
		FS: fstest.MapFS{
			"path/to/readable/dir": directory(0444),
		},
		Path: "path/to/readable/dir",
		Want: nil,
	},
	"path to unreadable dir is invalid": {
		FS: fstest.MapFS{
			"path/to/unreadable/dir": directory(0333),
		},
		Path: "path/to/unreadable/dir",
		Want: ErrCannotRead,
	},
	"path to nowhere is invalid": {
		FS: fstest.MapFS{
			"path/to/nowhere": nil,
		},
		Path: "path/to/nowhere",
		Want: ErrNotExist,
	},
	"path to link is invalid": {
		FS: fstest.MapFS{
			"path/to/link": linkTo("some/place"),
		},
		Path: "path/to/link",
		Want: ErrNotDir,
	},
	"path to file is invalid": {
		FS: fstest.MapFS{
			"path/to/file": regularFile(),
		},
		Path: "path/to/file",
		Want: ErrNotDir,
	},
	"path to duffel dir is invalid": {
		FS: fstest.MapFS{
			"path/to/duffel-dir":         directory(0755),
			"path/to/duffel-dir/.duffel": regularFile(),
		},
		Path: "path/to/duffel-dir",
		Want: ErrIsDuffelDir,
	},
	"path to dir inside duffel is invalid": {
		FS: fstest.MapFS{
			"path/to/duffel-dir/.duffel":                       regularFile(),
			"path/to/duffel-dir/dir/dir/dir-inside-duffel-dir": directory(0755),
		},
		Path: "path/to/duffel-dir/dir/dir/dir-inside-duffel-dir",
		Want: ErrIsDuffelDir,
	},
}

func TestCheckInstallPath(t *testing.T) {
	for name, test := range installPathTests {
		t.Run(name, func(t *testing.T) {
			got := CheckInstallPath(test.FS, test.Path)
			if !errors.Is(got, test.Want) {
				t.Errorf("got error %v, want %v", got, test.Want)
			}
		})
	}
}
