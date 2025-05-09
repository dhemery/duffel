package rules

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

type packagePathRuleTest struct {
	FS   fs.FS
	Path string
	Want error
}

var packagePathRuleTests = map[string]packagePathRuleTest{
	"path to readable dir is valid": {
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
	"path to nowhere is invalid": {
		FS: fstest.MapFS{
			"path/to/nowhere": nil,
		},
		Path: "path/to/nowhere",
		Want: ErrNotExist,
	},
}

func TestPackagePathRules(t *testing.T) {
	for name, test := range packagePathRuleTests {
		t.Run(name, func(t *testing.T) {
			got := CheckPackagePath(test.FS, test.Path)
			if !errors.Is(got, test.Want) {
				t.Errorf("got error %v, want %v", got, test.Want)
			}
		})
	}
}
