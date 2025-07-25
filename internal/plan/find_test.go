package plan

import (
	"errors"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestPkgFinder(t *testing.T) {
	tests := map[string]struct {
		findName    string  // The name of the directory whose package info to find.
		duffelFile  string  // The path to the duffel file.
		wantPkgItem PkgItem // The the package item desired from FindPkg.
		wantErr     error   // The error desired from FindPkg.
	}{
		"not in a package": {
			findName:   "dir1/dir2/dir3/dir4",
			duffelFile: "",
			wantErr:    ErrNotInPackage,
		},
		"is a duffel source dir": {
			findName:   "dir1/dir2/dir3/dir4",
			duffelFile: "dir1/dir2/dir3/dir4/.duffel",
			wantErr:    ErrIsSource,
		},
		"is a duffel package": {
			duffelFile: "user/home/source/.duffel",
			findName:   "user/home/source/pkg",
			wantErr:    ErrIsPackage,
		},
		"in a duffel dir": {
			duffelFile: "user/home/source/.duffel",
			findName:   "user/home/source/pkg/item",
			wantPkgItem: PkgItem{
				Source: "user/home/source",
				Pkg:    "pkg",
				Item:   "item",
			},
			wantErr: nil,
		},
		"deep in a duffel dir": {
			findName:   "user/home/source/pkg/item1/item2/item3",
			duffelFile: "user/home/source/.duffel",
			wantPkgItem: PkgItem{
				Source: "user/home/source",
				Pkg:    "pkg",
				Item:   "item1/item2/item3",
			},
			wantErr: nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			errfs.AddDir(testFS, test.findName, 0o755)
			if test.duffelFile != "" {
				errfs.AddFile(testFS, test.duffelFile, 0o644)
			}

			finder := NewPkgFinder(testFS)

			gotPkgItem, gotErr := finder.FindPkg(test.findName)

			if gotPkgItem != test.wantPkgItem {
				t.Errorf("FindPkg(%q) result:\n got: %v\nwant: %v",
					test.findName, gotPkgItem, test.wantPkgItem)
			}
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("FindPkg(%q) error:\n got: %v\nwant %v",
					test.findName, gotErr, test.wantErr)
			}
		})
	}
}
