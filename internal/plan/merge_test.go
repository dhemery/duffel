package plan

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

type pkgFinderFunc func(name string) (PkgOp, error)

func (f pkgFinderFunc) FindPkg(name string) (PkgOp, error) {
	return f(name)
}

type analyzerFunc func(op PkgOp) error

func (f analyzerFunc) Analyze(op PkgOp) error {
	return f(op)
}

type testPkgOp struct {
	origin string
}

func (o testPkgOp) WalkDir() string {
	panic("unimplemented")
}

func (o testPkgOp) VisitFunc(Index) fs.WalkDirFunc {
	panic("unimplemented")
}

func TestMerge(t *testing.T) {
	aPkgFinderError := errors.New("error from packager")
	anAnalyzerError := errors.New("error from packager")

	tests := map[string]struct {
		pkgOp      PkgOp // PkgOp from pkg finder
		findErr    error // Error from pkg finder
		analyzeErr error // Error from analyzer
		wantErr    error // Error desired from Merge
	}{
		"package finder error": {
			findErr: aPkgFinderError,
			wantErr: aPkgFinderError,
		},
		"plan error": {
			pkgOp:      testPkgOp{"from plan error test"},
			findErr:    nil,
			analyzeErr: anAnalyzerError,
			wantErr:    anAnalyzerError,
		},
		"success": {
			pkgOp:      testPkgOp{"from success test"},
			findErr:    nil,
			analyzeErr: nil,
			wantErr:    nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			name := "file/name/to/find"

			var gotFindPkg bool
			pkgFinder := pkgFinderFunc(func(gotName string) (PkgOp, error) {
				gotFindPkg = true
				if gotName != name {
					t.Errorf("FindPkg(name) called with %q, want %q", gotName, name)
				}
				return test.pkgOp, test.findErr
			})

			var gotPlan bool
			analyzer := analyzerFunc(func(op PkgOp) error {
				gotPlan = true
				if op != test.pkgOp {
					t.Errorf("Plan(op) op arg:\n got: %v\nwant %v", op, test.pkgOp)
				}

				return test.analyzeErr
			})

			merger := NewMerger(pkgFinder, analyzer)

			err := merger.Merge(name)

			if !gotFindPkg {
				t.Errorf("Find(pkg) called: got %t, want %t", gotFindPkg, true)
			}

			wantPlan := test.findErr == nil
			if gotPlan != wantPlan {
				t.Errorf("Plan(pkgOp) called: got %t, want %t", gotPlan, wantPlan)
			}

			if !errors.Is(err, test.wantErr) {
				t.Errorf("error: got %v, want %v", err, test.wantErr)
			}
		})
	}
}

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
