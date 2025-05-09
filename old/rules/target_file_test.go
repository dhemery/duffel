package rules

import (
	"errors"
	"testing"
	"testing/fstest"
)

type targetPathTest struct {
	FS          fstest.MapFS
	Path        string
	WantCanLink bool
	WantError   error
}

// TEST IDEAS
//
// Target Path Rules
// - dir: target dir rules
// - link: target link rules
//
// Target Dir Rules
// - is farm: err invalid
// - source dir: continue walking
//	what did I mean by this?
//
// Target Link Rules
// - To source
// - To farm
// - To package
// - Into current package
// - Into local package
// - Into foreign farm
// - Into foreign package

var targetPathTests = map[string]targetPathTest{
	"path to nowhere can link": {
		FS: fstest.MapFS{
			"path/to/nowhere": nil,
		},
		Path:        "path/to/nowhere",
		WantCanLink: true,
		WantError:   nil,
	},
	"path to file is invalid": {
		FS: fstest.MapFS{
			"path/to/file": regularFile(),
		},
		Path:        "path/to/file",
		WantCanLink: false,
		WantError:   ErrIsFile,
	},
}

func TestCheckTargetPath(t *testing.T) {
	for name, test := range targetPathTests {
		t.Run(name, func(t *testing.T) {
			canLink, err := CheckTargetPath(test.FS, test.Path)
			if canLink != test.WantCanLink {
				t.Errorf("got can link %t, want %t", canLink, test.WantCanLink)
			}
			if !errors.Is(err, test.WantError) {
				t.Errorf("got error %v, want %v", err, test.WantError)
			}

		})
	}

}
