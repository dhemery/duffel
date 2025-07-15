package plan

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/file"

	"github.com/google/go-cmp/cmp"
)

const (
	target = "path/to/target"
	source = "path/to/source"
	pkg    = "pkg"
)

var targetToSource, _ = filepath.Rel(target, source)

type testDirEntry struct {
	name string
	mode fs.FileMode
}

func (e testDirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}

func (e testDirEntry) IsDir() bool {
	return e.mode.IsDir()
}

func (e testDirEntry) Name() string {
	return e.name
}

func (e testDirEntry) Type() fs.FileMode {
	return e.mode.Type()
}

func TestInstallOp(t *testing.T) {
	tests := map[string]struct {
		item        string      // Item being analyzed, relative to pkg dir
		entry       fs.DirEntry // Dir entry passed to Apply for the item
		targetState *file.State // Target state passed to Apply
		wantState   *file.State // State returned by Apply
		wantErr     error       // Error returned by Apply
	}{
		"create new target link to dir item": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: nil,
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		"create new target link to non-dir item": {
			item:        "item",
			targetState: nil,
			entry:       testDirEntry{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"create new target link to sub-item": {
			item:        "dir/sub1/sub2/item",
			targetState: nil,
			entry:       testDirEntry{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
		"install dir item contents to existing target dir": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
			// No change in state
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// No error, so walk will continue with the item's contents
			wantErr: nil,
		},
		"target already links to current dir item": {
			item:  "item",
			entry: testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			// Do not walk the dir item. It's already linked.
			wantErr: fs.SkipDir,
		},
		"target already links to current non-dir item": {
			item:  "item",
			entry: testDirEntry{mode: 0o644},
			targetState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"target already links to current sub-item": {
			item:  "dir/sub1/sub2/item",
			entry: testDirEntry{mode: 0o644},
			targetState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(source, target, nil, nil)

			gotState, gotErr := install.Apply(pkg, test.item, test.entry, test.targetState)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply() error:\n got %v\nwant %v", gotErr, test.wantErr)
			}

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("Apply() state result:\n got %#v\nwant %#v", gotState, test.wantState)
			}
		})
	}
}

func TestInstallOpConlictErrors(t *testing.T) {
	tests := map[string]struct {
		sourceEntry fs.DirEntry // The dir entry for the item
		targetState *file.State // The existing target state for the item
	}{
		"target is a file, source is a dir": {
			sourceEntry: testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: 0o644},
		},
		"target is unknown type, source is a dir": {
			sourceEntry: testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDevice},
		},
		"target links to a non-dir, source is a dir": {
			sourceEntry: testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "link/to/file", DestMode: 0o644},
		},
		"target is a dir, source is not a dir": {
			sourceEntry: testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"target links to a dir, source is not a dir": {
			sourceEntry: testDirEntry{mode: 0o644},
			targetState: &file.State{
				Mode:     fs.ModeSymlink,
				Dest:     "target/some/dest",
				DestMode: fs.ModeDir | 0o755,
			},
		},
	}

	for name, test := range tests {
		const item = "item"
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(source, target, nil, nil)

			gotState, gotErr := install.Apply(pkg, item, test.sourceEntry, test.targetState)

			if gotState != nil {
				t.Errorf("Apply() state: want nil, got %v", gotState)
			}

			var wantErr *InstallError

			if !errors.As(gotErr, &wantErr) {
				t.Errorf("Apply() error:\n got %s\nwant *InstallError", gotErr)
			}
		})
	}
}

type testAnalyzer struct {
	gotOp PkgOp
	err   error
}

func (a *testAnalyzer) Analyze(op PkgOp) error {
	a.gotOp = op
	return a.err
}

type testPkgFinder struct {
	gotName string
	result  string
	err     error
}

func (pf *testPkgFinder) FindPkg(name string) (string, error) {
	pf.gotName = name
	return pf.result, pf.err
}

// If the package item is a dir
// and the target is a link to a dir in a duffel package,
// install should replace the target link with a dir
// and analyze the linked dir.
func TestInstallOpMerge(t *testing.T) {
	TestAnalyzeError := errors.New("test error returned from Analyze")
	TestFindPkgError := errors.New("test error returned from FindPkg")
	TestUnepectedAnalyzeCall := errors.New("unexpected call to Analyze")

	tests := map[string]struct {
		dest             string
		findPkgResult    string
		findErrResult    error
		analyzeErrResult error
		wantState        *file.State
		wantErr          error
	}{
		"find pkg returns error": {
			dest:             "../../some/non/pkg/item",
			findPkgResult:    "",
			findErrResult:    TestFindPkgError,
			analyzeErrResult: TestUnepectedAnalyzeCall,
			wantState:        nil,
			wantErr:          TestFindPkgError,
		},
		"analyze returns error": {
			dest:             "../../some/foreign/pkg/an/item",
			findPkgResult:    path.Join(target, "../../some/foreign/pkg"),
			findErrResult:    nil,
			analyzeErrResult: TestAnalyzeError,
			wantState:        nil,
			wantErr:          TestAnalyzeError,
		},
		"analyzes foreign package": {
			dest:             "../../some/foreign/pkg/an/item",
			findPkgResult:    path.Join(target, "../../some/foreign/pkg"),
			findErrResult:    nil,
			analyzeErrResult: nil,
			// On merge success, replace the target link with a dir
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// Walk the current package item's contents
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			pkg := "pkg"
			item := "item"

			// Install merges only if the package item is a dir
			entry := testDirEntry{name: item, mode: fs.ModeDir | 0o755}

			// Install merges only if the target is a link to a dir
			state := &file.State{Mode: fs.ModeSymlink, Dest: test.dest, DestMode: fs.ModeDir | 0o755}

			testPkgFinder := testPkgFinder{
				result: test.findPkgResult,
				err:    test.findErrResult,
			}

			testAnalyzer := testAnalyzer{err: test.analyzeErrResult}

			install := NewInstallOp(source, target, &testPkgFinder, &testAnalyzer)

			gotState, gotErr := install.Apply(pkg, item, entry, state)

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("Apply() state result:\n got %v\nwant %v", gotState, test.wantState)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply() error:\n got %v\nwant %v", gotErr, test.wantErr)
			}

			wantFindName := path.Join(target, test.dest)
			if !cmp.Equal(testPkgFinder.gotName, wantFindName) {
				t.Errorf("FindPkg() name: got %q, want %q", testPkgFinder.gotName, wantFindName)
			}

			var wantAnalyzeOp PkgOp
			if test.findErrResult == nil {
				wantWalkDir := path.Join(target, test.dest)
				wantAnalyzeOp = NewForeignPkgOp(test.findPkgResult, wantWalkDir, install)
			}
			if !cmp.Equal(testAnalyzer.gotOp, wantAnalyzeOp) {
				t.Errorf("Analyze() op:\n got %q\nwant %q", testAnalyzer.gotOp, wantAnalyzeOp)
			}
		})
	}
}
