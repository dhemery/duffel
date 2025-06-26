package plan

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
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
		entry       fs.DirEntry // Pkg item entry passed to Apply
		targetState *file.State // Target state passed to Apply
		wantState   *file.State // State returned by by Apply
		wantErr     error       // Error returned by Apply
	}{
		"no target state, source is dir": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: nil,
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		"no target state, source is sub-item": {
			item:        "dir/sub1/sub2/item",
			targetState: nil,
			entry:       testDirEntry{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
		"no target state, source is non-dir": {
			item:        "item",
			targetState: nil,
			entry:       testDirEntry{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"target is dir, source is dir": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
			// No change in state
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// No error, so walk will continue with pkg item's contents
			wantErr: nil,
		},
		"target links to current item dir": {
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
			// Do not walk the dir. It's already linked.
			wantErr: fs.SkipDir,
		},
		"target links to current item": {
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
		"target links to current sub-item": {
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
			install := Install{
				TargetToSource: targetToSource,
			}

			gotState, gotErr := install.Apply(pkg, test.item, test.entry, test.targetState)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			if !reflect.DeepEqual(gotState, test.wantState) {
				t.Errorf("state result:\nwant %#v\ngot  %#v", test.wantState, gotState)
			}
		})
	}
}

func TestInstallOpConflictError(t *testing.T) {
	tests := map[string]struct {
		sourceEntry fs.DirEntry
		targetState *file.State
	}{
		"target is dir, source is not dir": {
			sourceEntry: testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"target is link, source is not dir": {
			sourceEntry: testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "target/some/dest"},
		},
	}
	targetToSource, _ := filepath.Rel(target, source)

	for name, test := range tests {
		const item = "item"
		t.Run(name, func(t *testing.T) {
			install := Install{
				TargetToSource: targetToSource,
			}

			gotState, gotErr := install.Apply(pkg, item, test.sourceEntry, test.targetState)

			if gotState != nil {
				t.Errorf("state: want nil, got %v", gotState)
			}

			wantErr := &ErrConflict{
				Op:          "install",
				Pkg:         pkg,
				Item:        item,
				SourceType:  test.sourceEntry.Type(),
				TargetState: test.targetState,
			}
			if !reflect.DeepEqual(gotErr, wantErr) {
				t.Errorf("error:\nwant %#v\n got %#v", wantErr, gotErr)
			}
		})
	}
}

func TestInstallOpInvalidTarget(t *testing.T) {
	tests := map[string]struct {
		targetState *file.State
		wantErr     error
		skip        string
	}{
		"target is file": {
			targetState: &file.State{Mode: 0o644},
			wantErr:     ErrIsFile,
		},
		"target is unknown type": {
			targetState: &file.State{Mode: fs.ModeDevice},
			wantErr:     ErrUnknownType,
		},
		"target links to foreign dest": {
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "target/foreign/dest"},
			wantErr:     ErrDestNotPkgItem,
			skip:        "not yet implemented",
		},
	}
	targetToSource, _ := filepath.Rel(target, source)

	for name, test := range tests {
		const item = "item"
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}
			install := Install{
				TargetToSource: targetToSource,
			}

			gotState, gotErr := install.Apply(pkg, item, testDirEntry{mode: 0o644}, test.targetState)

			if gotState != nil {
				t.Errorf("state: want nil, got %v", gotState)
			}

			wantErr := &ErrInvalidTarget{
				Op:    "install",
				Pkg:   pkg,
				Item:  item,
				State: test.targetState,
				Err:   test.wantErr,
			}
			if !reflect.DeepEqual(gotErr, wantErr) {
				t.Errorf("error:\nwant %#v\n got %#v", wantErr, gotErr)
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
