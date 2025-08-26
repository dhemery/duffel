package plan

import (
	"errors"
	"path"
	"testing"

	"github.com/dhemery/duffel/internal/file"

	"github.com/dhemery/duffel/internal/errfs"
)

func TestItemizer(t *testing.T) {
	tests := map[string]struct {
		sourceDir      string     // The duffel source dir in the file system.
		nameArg        string     // The name passed to Itemize.
		wantSourcePath sourcePath // The SourcePath result from Itemize.
		wantErr        error      // The error result from Itemize.
	}{
		"not in a source dir": {
			sourceDir: "elsewhere",
			nameArg:   "dir1/dir2/dir3/dir4",
			wantErr:   errNotInPackage,
		},
		"source dir": {
			sourceDir: "dir1/dir2/dir3/dir4",
			nameArg:   "dir1/dir2/dir3/dir4",
			wantErr:   errIsSource,
		},
		"package": {
			sourceDir: "user/home/source",
			nameArg:   "user/home/source/pkg",
			wantErr:   errIsPackage,
		},
		"child of a package": {
			sourceDir:      "user/home/source",
			nameArg:        "user/home/source/pkg/item",
			wantSourcePath: newSourcePath("user/home/source", "pkg", "item"),
		},
		"deep in a package": {
			sourceDir:      "user/home/source",
			nameArg:        "user/home/source/pkg/item1/item2/item3",
			wantSourcePath: newSourcePath("user/home/source", "pkg", "item1/item2/item3"),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			errfs.AddDir(testFS, test.nameArg, 0o755)
			if test.sourceDir != "" {
				errfs.AddFile(testFS, path.Join(test.sourceDir, file.SourceMarkerFile), 0o644)
			}

			itemizer := itemizer{testFS}

			gotPackageItem, gotErr := itemizer.itemize(test.nameArg)

			if gotPackageItem != test.wantSourcePath {
				t.Errorf("Itemize(%q) result:\n got: %v\nwant: %v",
					test.nameArg, gotPackageItem, test.wantSourcePath)
			}
			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Itemize(%q) error:\n got: %v\nwant %v",
					test.nameArg, gotErr, test.wantErr)
			}
		})
	}
}
