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
		"target item is dir": {
			item:        "item",
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
			wantErr: fs.SkipDir, // Do not walk the dir. It's already linked.
		},
		"target item links to current pkg item non-dir": {
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
		"target item links to foreign dest": {
			item:        "item",
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "current/foreign/dest"},
			wantState:   nil,
			wantErr:     ErrNotPkgItem,
		},
		"target item is not file, dir, or link": {
			item:        "item",
			targetState: &file.State{Mode: fs.ModeDevice},
			wantErr:     ErrTargetType,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := Install{
				TargetToSource: targetToSource,
			}

			gotAdvice, gotErr := install.Apply(pkg, test.item, test.entry, test.targetState)

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}

			if !reflect.DeepEqual(gotAdvice, test.wantState) {
				t.Errorf("item advice:\nwant %#v\ngot  %#v", test.wantState, gotAdvice)
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
