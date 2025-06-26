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

func TestInstallOp(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item        string      // Item being analyzed, relative to pkg dir
		entry       fs.DirEntry // Pkg item entry passed to Apply
		targetState *file.State // Target state passed to Apply
		wantState   *file.State // State returned by by Apply
		wantErr     error       // Error returned by Apply
	}{
		"no target state, item is dir": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: nil,
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: fs.SkipDir, // Do not walk the dir. Linking to it suffices.
		},
		"no target state, item is sub-item": {
			item:        "dir/sub1/sub2/item",
			targetState: nil,
			entry:       testDirEntry{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
		"no target state, item is non-dir": {
			item:        "item",
			targetState: nil,
			entry:       testDirEntry{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"target item is dir, pkg item is dir": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
			// No change in state
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// No error, so walk will continue with pkg item's contents
			wantErr: nil,
		},
		"target item is dir, pkg item is non-dir": {
			item:        "item",
			entry:       testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
			wantErr:     ErrIsDir,
		},
		"target item is file": {
			item:        "item",
			targetState: &file.State{Mode: 0o644},
			wantErr:     ErrIsFile,
		},
		"target item links to current pkg item dir": {
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
		"target item links to current item": {
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
		"target item links to current sub-item": {
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
		"target is link, pkg item is not dir": {
			item:        "item",
			entry:       testDirEntry{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "target/some/dest"},
			wantState:   nil,
			wantErr:     ErrNotDir,
		},
		"target item links to foreign dest": {
			item:        "item",
			entry:       testDirEntry{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "target/foreign/dest"},
			wantState:   nil,
			wantErr:     ErrNotPkgItem,
		},
		"target item is not file, dir, or link": {
			item:        "item",
			targetState: &file.State{Mode: fs.ModeDevice},
			wantErr:     ErrUnknownType,
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

type testDirEntry struct {
	name string
	mode fs.FileMode
}

func (e testDirEntry) Info() (fs.FileInfo, error) {
	return nil, nil
}

func (e testDirEntry) IsDir() bool {
	return e.Mode().IsDir()
}

func (e testDirEntry) Mode() fs.FileMode {
	return e.mode
}

func (e testDirEntry) Name() string {
	return e.name
}

func (e testDirEntry) Type() fs.FileMode {
	return e.Mode().Type()
}
