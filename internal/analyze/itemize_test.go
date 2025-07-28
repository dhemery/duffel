package analyze_test

import (
	"errors"
	"testing"

	. "github.com/dhemery/duffel/internal/analyze"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestItemizer(t *testing.T) {
	tests := map[string]struct {
		findName        string      // The name of the directory whose path to itemize.
		duffelFile      string      // The path to the duffel file.
		wantPackageItem PackageItem // The the package item desired from Itemize.
		wantErr         error       // The error desired from Itemize.
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
			wantPackageItem: PackageItem{
				Source:  "user/home/source",
				Package: "pkg",
				Item:    "item",
			},
			wantErr: nil,
		},
		"deep in a duffel dir": {
			findName:   "user/home/source/pkg/item1/item2/item3",
			duffelFile: "user/home/source/.duffel",
			wantPackageItem: PackageItem{
				Source:  "user/home/source",
				Package: "pkg",
				Item:    "item1/item2/item3",
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

			itemizer := NewItemizer(testFS)

			gotPackageItem, gotErr := itemizer.Itemize(test.findName)

			if gotPackageItem != test.wantPackageItem {
				t.Errorf("Itemize(%q) result:\n got: %v\nwant: %v",
					test.findName, gotPackageItem, test.wantPackageItem)
			}
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Itemize(%q) error:\n got: %v\nwant %v",
					test.findName, gotErr, test.wantErr)
			}
		})
	}
}
