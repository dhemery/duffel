package plan

import (
	"errors"
	"io/fs"
	"testing"
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
