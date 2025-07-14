package plan

import (
	"errors"
	"io/fs"
	"slices"
	"testing"
)

type pkgFinderFunc func(name string) (PkgOp, error)

func (f pkgFinderFunc) FindPkg(name string) (PkgOp, error) {
	return f(name)
}

type plannerFunc func(ops ...PkgOp) error

func (f plannerFunc) Plan(ops ...PkgOp) error {
	return f(ops...)
}

type testPkgOp struct {
	origin string
}

func (o testPkgOp) WalkDir() string {
	panic("unimplemented")
}

func (o testPkgOp) VisitFunc() fs.WalkDirFunc {
	panic("unimplemented")
}

func TestMerge(t *testing.T) {
	aPkgFinderError := errors.New("error from packager")
	aPlannerError := errors.New("error from packager")

	tests := map[string]struct {
		pkgOp   PkgOp // PkgOp from pkg finder
		findErr error // Error from pkg finder
		planErr error // Error from planner
		wantErr error // Error desired from Merge
	}{
		"package finder error": {
			findErr: aPkgFinderError,
			wantErr: aPkgFinderError,
		},
		"plan error": {
			pkgOp:   testPkgOp{"from plan error test"},
			findErr: nil,
			planErr: aPlannerError,
			wantErr: aPlannerError,
		},
		"success": {
			pkgOp:   testPkgOp{"from success test"},
			findErr: nil,
			planErr: nil,
			wantErr: nil,
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
			planner := plannerFunc(func(gotOps ...PkgOp) error {
				gotPlan = true
				wantOps := []PkgOp{test.pkgOp}
				if !slices.Equal(gotOps, wantOps) {
					t.Errorf("Plan(ops) ops arg:\n got: %v\nwant %v", gotOps, wantOps)
				}

				return test.planErr
			})

			merger := NewMerger(pkgFinder, planner)

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
