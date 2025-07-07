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
			install := Install{Source: source, Target: target}

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
			install := Install{Source: source, Target: target}

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

type mergeFunc func(name string) error

func (f mergeFunc) Merge(name string) error {
	return f(name)
}

// If the packge item is a dir and the target is a link to a dir,
// Install should call Merge with the target's destination.
func TestInstallOpMerge(t *testing.T) {
	aMergeError := errors.New("error returned from Merge")

	tests := map[string]struct {
		mergeErr  error
		wantState *file.State
		wantErr   error
	}{
		"success": {
			mergeErr: nil,
			// On merge success, replace the target link with a dir
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// Walk the package item's contents
			wantErr: nil,
		},
		"merge error": {
			mergeErr:  aMergeError,
			wantState: nil,
			wantErr:   aMergeError,
		},
	}

	for name, test := range tests {
		const (
			item = "item"
			dest = "link/to/dest"
		)

		t.Run(name, func(t *testing.T) {
			// Install merges only if the package item is a dir
			entry := testDirEntry{name: item, mode: fs.ModeDir | 0o755}
			// Install merges only if the target is a link to a dir
			state := &file.State{Mode: fs.ModeSymlink, Dest: dest, DestMode: fs.ModeDir | 0o755}

			wantMergeName := path.Join(target, dest)

			var mergeCalled bool
			merger := mergeFunc(func(gotName string) error {
				t.Helper()
				mergeCalled = true
				if gotName != wantMergeName {
					t.Errorf("Merge() called with %q, want %q", gotName, wantMergeName)
				}

				return test.mergeErr
			})

			install := Install{Source: source, Target: target, Merger: merger}

			gotState, gotErr := install.Apply(pkg, item, entry, state)

			if !mergeCalled {
				t.Errorf("Merge() not called")
			}

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("Apply() state result:\n got %#v\nwant %#v", gotState, test.wantState)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply() error:\n got %v\nwant %v", gotErr, test.wantErr)
			}
		})
	}
}

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
