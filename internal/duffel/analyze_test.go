package duffel

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/duftest"
)

type itemAnalystFunc func(pkg, item string, d fs.DirEntry) error

func (tia itemAnalystFunc) Analyze(pkg, item string, d fs.DirEntry) error {
	return tia(pkg, item, d)
}

func TestPkgAnalyst(t *testing.T) {
	const (
		target         = "path/to/target"
		source         = "path/to/source"
		pkg            = "pkg"
		item           = "item"
		permReadable   = 0o444
		permUnreadable = 0
	)

	type analyzeItemCall struct {
		pkgArg  string
		itemArg string
		err     error
	}
	tests := map[string]struct {
		files           fstest.MapFS     // Files on the file system
		wantAnalyzeItem *analyzeItemCall // Wanted call to item analyzer
		wantErr         error            // Error returned by pkg analyzer
	}{
		"readable pkg dir": {
			files: fstest.MapFS{
				path.Join(source, pkg): duftest.DirEntry(permReadable),
			},
			wantErr: nil,
		},
		"unreadable pkg dir": {
			files: fstest.MapFS{
				path.Join(source, pkg): duftest.DirEntry(permUnreadable),
			},
			wantErr: fs.ErrPermission,
		},
		"readable pkg item": {
			files: fstest.MapFS{
				path.Join(source, pkg, item): duftest.FileEntry("", permReadable),
			},
			wantAnalyzeItem: &analyzeItemCall{
				pkgArg:  pkg,
				itemArg: item,
				err:     nil,
			},
			wantErr: nil,
		},
		"unreadable dir item": {
			files: fstest.MapFS{
				path.Join(source, pkg, item): duftest.DirEntry(permUnreadable),
			},
			// Called for item's dir entry before trying to read item itself.
			wantAnalyzeItem: &analyzeItemCall{
				pkgArg:  pkg,
				itemArg: item,
				err:     nil,
			},
			wantErr: fs.ErrPermission,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotAnalyzeItemCall := false

			tia := itemAnalystFunc(func(pkg, item string, d fs.DirEntry) error {
				if gotAnalyzeItemCall {
					return fmt.Errorf("analyze item: unexpected second call: pkg %q, item %q",
						pkg, item)
				}
				want := test.wantAnalyzeItem
				if want == nil {
					return fmt.Errorf("analyze item: unexpected call with pkg %q, item %q",
						pkg, item)
				}
				gotAnalyzeItemCall = true
				if pkg != want.pkgArg {
					t.Errorf("analyze item: want pkg %q, got %q", want.pkgArg, pkg)
				}
				if item != want.itemArg {
					t.Errorf("analyze item:, want item %q, got %q", want.itemArg, item)
				}
				return want.err
			})

			fsys := duftest.FS{M: test.files}

			pa := NewPkgAnalyst(fsys, source, pkg, tia)

			gotErr := pa.Analyze()

			if test.wantAnalyzeItem != nil && !gotAnalyzeItemCall {
				t.Errorf("no call to analyze item, wanted: %#v", test.wantAnalyzeItem)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}
		})
	}
}
