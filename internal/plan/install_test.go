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
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("state result:\nwant %#v\ngot  %#v", test.wantState, gotState)
			}
		})
	}
}

func TestInstallOpConflictError(t *testing.T) {
	tests := map[string]struct {
		sourceEntry fs.DirEntry // The dir entry for the item
		targetState *file.State // The existing target state for the item
	}{
		"target is a dir, source is not a dir": {
			sourceEntry: testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"target links to a dir, source is not a dir": {
			sourceEntry: testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "target/some/dest", DestMode: fs.ModeDir | 0o755},
		},
	}

	for name, test := range tests {
		const item = "item"
		t.Run(name, func(t *testing.T) {
			install := Install{Source: source, Target: target}

			gotState, gotErr := install.Apply(pkg, item, test.sourceEntry, test.targetState)

			if gotState != nil {
				t.Errorf("state: want nil, got %v", gotState)
			}

			wantErr := &ConflictError{
				Op:          "install",
				Pkg:         pkg,
				Item:        item,
				ItemType:    test.sourceEntry.Type(),
				TargetState: test.targetState,
			}
			if gotErr.Error() != wantErr.Error() {
				t.Errorf("error:\nwant %s\n got %s", wantErr, gotErr)
			}
		})
	}
}

func TestInstallOpInvalidTarget(t *testing.T) {
	tests := map[string]struct {
		targetState *file.State // The invalid target state
		skip        string      // The reason to skip this test
	}{
		"target is a file": {
			targetState: &file.State{Mode: 0o644},
		},
		"target is unknown type": {
			targetState: &file.State{Mode: fs.ModeDevice},
		},
		"target links to a non-dir": {
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "link/to/file", DestMode: 0o644},
		},
	}

	for name, test := range tests {
		const item = "item"
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			install := Install{Source: source, Target: target}

			gotState, gotErr := install.Apply(pkg, item, nil, test.targetState)

			if gotState != nil {
				t.Errorf("state: want nil, got %v", gotState)
			}

			wantErr := &TargetError{
				Op:    "install",
				Pkg:   pkg,
				Item:  item,
				State: test.targetState,
			}
			if gotErr.Error() != wantErr.Error() {
				t.Errorf("error:\nwant %s\n got %s", wantErr, gotErr)
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
