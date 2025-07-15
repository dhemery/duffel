package file

import (
	"errors"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestPkgFinder(t *testing.T) {
	tests := map[string]struct {
		dest        string // The link destination whose package to find.
		duffelFile  string // The path to the duffel file.
		wantPkgPath string // The the package path desired from FindPkg.
		wantErr     error  // The error desired from FindPkg.
	}{
		"not in a package": {
			dest:       "dir1/dir2/dir3/dir4",
			duffelFile: "",
			wantErr:    ErrNotInPackage,
		},
		"is a duffel source dir": {
			dest:       "dir1/dir2/dir3/dir4",
			duffelFile: "dir1/dir2/dir3/dir4/.duffel",
			wantErr:    ErrIsSource,
		},
		"is a duffel package": {
			duffelFile: "user/home/source/.duffel",
			dest:       "user/home/source/pkg",
			wantErr:    ErrIsPackage,
		},
		"in a duffel dir": {
			duffelFile:  "user/home/source/.duffel",
			dest:        "user/home/source/pkg/item",
			wantPkgPath: "user/home/source/pkg",
			wantErr:     nil,
		},
		"deep in a duffel dir": {
			dest:        "user/home/source/pkg/item1/item2/item3",
			duffelFile:  "user/home/source/.duffel",
			wantPkgPath: "user/home/source/pkg",
			wantErr:     nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			testFS.AddDir(test.dest, 0o755)
			if test.duffelFile != "" {
				testFS.AddFile(test.duffelFile, 0o644)
			}

			finder := NewPkgFinder(testFS)

			gotPkgPath, gotErr := finder.FindPkg(test.dest)

			if gotPkgPath != test.wantPkgPath {
				t.Errorf("package path: got %q, want %q", gotPkgPath, test.wantPkgPath)
			}
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("err: got %v, want %v", gotErr, test.wantErr)
			}
		})
	}
}
