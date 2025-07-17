package plan

import (
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"

	"github.com/google/go-cmp/cmp"
)

type testFile struct {
	name string
	mode fs.FileMode
	dest string
}

func (tf testFile) Info() (fs.FileInfo, error) {
	return nil, nil
}

func (tf testFile) IsDir() bool {
	return tf.mode.IsDir()
}

func (tf testFile) Name() string {
	return tf.name
}

func (tf testFile) Type() fs.FileMode {
	return tf.mode.Type()
}

func TestInstallOp(t *testing.T) {
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
	)
	targetToSource, _ := filepath.Rel(target, source)

	tests := map[string]struct {
		item        string      // Item being analyzed, relative to pkg dir
		entry       fs.DirEntry // Dir entry passed to Apply for the item
		targetState *file.State // Target state passed to Apply
		wantState   *file.State // State returned by Apply
		wantErr     error       // Error returned by Apply
	}{
		"create new target link to dir item": {
			item:        "item",
			entry:       testFile{mode: fs.ModeDir | 0o755},
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
			entry:       testFile{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join(targetToSource, pkg, "item"),
			},
			wantErr: nil,
		},
		"create new target link to sub-item": {
			item:        "dir/sub1/sub2/item",
			targetState: nil,
			entry:       testFile{mode: 0o644},
			wantState: &file.State{
				Mode: fs.ModeSymlink,
				Dest: path.Join("..", "..", "..", targetToSource, pkg, "dir/sub1/sub2/item"),
			},
			wantErr: nil,
		},
		"install dir item contents to existing target dir": {
			item:        "item",
			entry:       testFile{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
			// No change in state
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// No error, so walk will continue with the item's contents
			wantErr: nil,
		},
		"target already links to current dir item": {
			item:  "item",
			entry: testFile{mode: fs.ModeDir | 0o755},
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
			entry: testFile{mode: 0o644},
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
			entry: testFile{mode: 0o644},
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
			install := NewInstallOp(source, target, nil)

			sourcePkgItem := path.Join(source, pkg, test.item)
			gotState, gotErr := install.Apply(sourcePkgItem, test.entry, test.targetState)

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
	const (
		target = "path/to/target"
		source = "path/to/source"
		pkg    = "pkg"
		item   = "item"
	)
	tests := map[string]struct {
		sourceEntry fs.DirEntry // The dir entry for the item
		targetState *file.State // The existing target state for the item
	}{
		"target is a file, source is a dir": {
			sourceEntry: testFile{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: 0o644},
		},
		"target is unknown type, source is a dir": {
			sourceEntry: testFile{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeDevice},
		},
		"target links to a non-dir, source is a dir": {
			sourceEntry: testFile{mode: fs.ModeDir | 0o755},
			targetState: &file.State{Mode: fs.ModeSymlink, Dest: "link/to/file", DestMode: 0o644},
		},
		"target is a dir, source is not a dir": {
			sourceEntry: testFile{mode: 0o644},
			targetState: &file.State{Mode: fs.ModeDir | 0o755},
		},
		"target links to a dir, source is not a dir": {
			sourceEntry: testFile{mode: 0o644},
			targetState: &file.State{
				Mode:     fs.ModeSymlink,
				Dest:     "target/some/dest",
				DestMode: fs.ModeDir | 0o755,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			install := NewInstallOp(source, target, nil)

			sourcePkgItem := path.Join(source, pkg, item)
			gotState, gotErr := install.Apply(sourcePkgItem, test.sourceEntry, test.targetState)

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

// If the package item is a dir
// and the target is a link to a dir in a duffel package,
// install should replace the target link with a dir
// and analyze the linked dir.
func TestRealInstallOpMerge(t *testing.T) {
	tests := map[string]struct {
		item           string                 // Tne name of the item being installed.
		target         string                 // The path to the target dir.
		targetItemDest string                 // The destination of the target symlink.
		files          []testFile             // Files to be analyzed for merging.
		wantState      *file.State            // State returned by Apply.
		wantErr        error                  // Error returned by Apply.
		wantIndex      map[string]*file.State // States added to index during Apply.
	}{
		"dest is not in a package": {
			item:           "item",
			target:         "target",
			targetItemDest: "../dir1/dir2/dir3/dir4/dir5/dir6",
			files: []testFile{{
				name: "dir1/dir2/dir3/dir4/dir5/dir6",
				mode: fs.ModeDir | 0o755,
			}},
			wantState: nil,
			wantErr:   file.ErrNotInPackage,
		},
		"dest is a duffel source dir": {
			item:           "item",
			target:         "target",
			targetItemDest: "../foreign/source-dir",
			files: []testFile{{
				name: "foreign/source-dir/.duffel",
				mode: 0o644,
			}},
			wantState: nil,
			wantErr:   file.ErrIsSource,
		},
		"dest is duffel package": {
			item:           "item",
			target:         "target",
			targetItemDest: "../foreign/source-dir/pkg-dir",
			files: []testFile{{
				name: "foreign/source-dir/.duffel",
				mode: 0o644,
			}, {
				name: "foreign/source-dir/pkg-dir/item/content",
				mode: 0o644,
			}},
			wantState: nil,
			wantErr:   file.ErrIsPackage,
		},
		"dest is a top level item in foreign package": {
			item:           "item",
			target:         "target",
			targetItemDest: "../foreign/source-dir/pkg-dir/item",
			files: []testFile{{
				name: "foreign/source-dir/.duffel",
				mode: 0o644,
			}, {
				name: "foreign/source-dir/pkg-dir/item/content",
				mode: 0o644,
			}},
			// Convert the existing target link into a dir.
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// Return nil to walk the current dir and install its contents into the new dir.
			wantErr: nil,
		},
		"dest is a nested item in a foreign package": {
			item:           "item",
			target:         "target",
			targetItemDest: "../foreign/source-dir/pkg-dir/item1/item2/item3",
			files: []testFile{{
				name: "foreign/source-dir/.duffel",
				mode: 0o644,
			}, {
				name: "foreign/source-dir/pkg-dir/item1/item2/item3/content",
				mode: 0o644,
			}},
			// Convert the existing target link into a dir.
			wantState: &file.State{Mode: fs.ModeDir | 0o755},
			// Return nil to walk the current dir and install its contents into the new dir.
			wantErr: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			const (
				source = "local/source"
				pkg    = "pkg"
			)
			testFS := errfs.New()
			for _, tf := range test.files {
				testFS.Add(tf.name, tf.mode, "")
			}
			pkgFinder := file.NewPkgFinder(testFS)
			stater := file.Stater{FS: testFS}
			index := NewIndex(stater)
			analyzer := NewAnalyst(testFS, index)
			merger := NewMerger(pkgFinder, analyzer)
			install := NewInstallOp(source, test.target, merger)

			// Install tries to merge only if the item entry is a dir.
			entry := testFile{name: path.Base(test.item), mode: fs.ModeDir | 0o755}
			// Install tries to merge only if the target item is a link to a dir.
			state := &file.State{Mode: fs.ModeSymlink, Dest: test.targetItemDest, DestMode: fs.ModeDir}

			sourcePkgItem := path.Join(source, pkg, test.item)
			gotState, gotErr := install.Apply(sourcePkgItem, entry, state)

			if !cmp.Equal(gotState, test.wantState) {
				t.Errorf("Apply() state result:\n got %v\nwant %v", gotState, test.wantState)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Errorf("Apply() error:\n got %v\nwant %v", gotErr, test.wantErr)
			}
		})
	}
}
