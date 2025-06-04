package duffel

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/testfs"
)

type testItemAnalyst func(pkg, item string, d fs.DirEntry) error

func (tia testItemAnalyst) Analyze(pkg, item string, d fs.DirEntry) error {
	return tia(pkg, item, d)
}

func TestPkgAnalyst(t *testing.T) {
	const (
		target           = "path/to/target"
		source           = "path/to/source"
		pkg              = "pkg"
		item             = "item"
		permR            = 0o444
		permW            = 0o222
		permX            = 0o111
		permReadable     = permR | permW | permX
		permUnreadable   = permW | permX
		permUnsearchable = permW | permR
	)

	tests := map[string]struct {
		files    fstest.MapFS // Files on the file system
		wantPkg  string       // pkg passed to item analyzer
		wantItem string       // item passed to item analyzer
		itemErr  error        // Error returned by item analyzer
		wantErr  error        // Error returned by pkg analyzer
		skip     string       // Reason for skipping this test
	}{
		"readable pkg dir": {
			files: fstest.MapFS{
				path.Join(source, pkg): testfs.DirEntry(permReadable),
			},
			wantErr: nil,
		},
		"unreadable pkg dir": {
			files: fstest.MapFS{
				path.Join(source, pkg): testfs.DirEntry(permUnreadable),
			},
			wantErr: fs.ErrPermission,
		},
		"readable pkg item": {
			files: fstest.MapFS{
				path.Join(source, pkg, item): testfs.FileEntry("", permReadable),
			},
			wantPkg:  pkg,
			wantItem: item,
			itemErr:  nil,
			wantErr:  nil,
		},
		"unreadable pkg item entry": {
			files: fstest.MapFS{
				path.Join(source, pkg):       testfs.DirEntry(permUnsearchable),
				path.Join(source, pkg, item): testfs.FileEntry("", permReadable),
			},
			wantErr: fs.ErrPermission,
			skip:    "don't know how to cause stat error on item",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.skip != "" {
				t.Skip(test.skip)
			}

			fsys := testfs.FS{M: test.files}
			tia := testItemAnalyst(func(pkg, item string, d fs.DirEntry) error {
				if test.wantPkg == "" {
					return fmt.Errorf("item analyze: unexpected call with pkg %q, item %q",
						pkg, item)
				}
				if pkg != test.wantPkg {
					t.Errorf("item analyze: want pkg %q, got %q", test.wantPkg, pkg)
				}
				if item != test.wantItem {
					t.Errorf("item analyze:, want item %q, got %q", test.wantItem, item)
				}
				return test.itemErr
			})

			pa := NewPkgAnalyst(fsys, source, pkg, tia)

			gotErr := pa.Analyze()

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}
		})
	}
}
