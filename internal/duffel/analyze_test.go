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

type analyzeItemFunc func(pkg, item string, d fs.DirEntry) error

func (aif analyzeItemFunc) Analyze(pkg, item string, d fs.DirEntry) error {
	return aif(pkg, item, d)
}

func (tia analyzeItemFunc) Visit(string, string, fs.DirEntry) error {
	panic("visit called on analyst")
}

func TestPkgAnalystAnalyze(t *testing.T) {
	const (
		target        = "path/to/target"
		source        = "path/to/source"
		pkg           = "pkg"
		item          = "item"
		dirReadable   = fs.ModeDir | 0o755
		dirUnreadable = fs.ModeDir | 0o311
		fileReadable  = 0o644
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
				path.Join(source, pkg): &fstest.MapFile{Mode: dirReadable},
			},
			wantErr: nil,
		},
		"unreadable pkg dir": {
			files: fstest.MapFS{
				path.Join(source, pkg): &fstest.MapFile{Mode: dirUnreadable},
			},
			wantErr: fs.ErrPermission,
		},
		"readable pkg item": {
			files: fstest.MapFS{
				path.Join(source, pkg, item): &fstest.MapFile{Mode: fileReadable},
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
				path.Join(source, pkg, item): &fstest.MapFile{Mode: dirUnreadable},
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

			tia := analyzeItemFunc(func(pkg, item string, d fs.DirEntry) error {
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

type itemVisitFunc func(pkg, item string, d fs.DirEntry) error

func (ivf itemVisitFunc) Visit(pkg, item string, d fs.DirEntry) error {
	return ivf(pkg, item, d)
}

func (ivf itemVisitFunc) Analyze(string, string, fs.DirEntry) error {
	panic("analyze called on visitor")
}

func TestPkgAnalystVisitPath(t *testing.T) {
	const (
		target        = "path/to/target"
		source        = "path/to/source"
		pkg           = "pkg"
		item          = "item"
		dirReadable   = fs.ModeDir | 0o755
		dirUnreadable = fs.ModeDir | 0o311
		fileReadable  = 0o644
	)

	customWalkError := errors.New("error passed to VisitPath")

	type itemVisit struct {
		pkgArg    string
		itemArg   string
		entryArg  fs.DirEntry
		gapArg    FileGap
		gapResult FileGap
		errResult error
	}
	tests := map[string]struct {
		files         fstest.MapFS // Files on the file system
		priorFileGap  *FileGap     // The recorded file gap for the path before VisitPath
		walkPath      string       // The path passed to VisitPath
		walkEntry     fs.DirEntry  // The dir entry passed to VisitPath
		walkErr       error        // The error passed to VisitPath
		wantFileGap   *FileGap     // The recorded file gap for the path after VisitPath
		wantItemVisit *itemVisit   // Wanted call to item visitor
		wantErr       error        // Error returned by VisitPath
	}{
		"pkg dir and walk error": {
			walkPath:      path.Join(source, pkg),
			walkErr:       customWalkError,
			wantFileGap:   nil,             // Do not record a file gap for the pkg dir
			wantItemVisit: nil,             // Do not visit the pkg dir as an item
			wantErr:       customWalkError, // Return the walk error
		},
		"pkg dir and no walk error": {
			walkPath:      path.Join(source, pkg),
			walkErr:       nil,
			wantItemVisit: nil, // Do not visit the pkg dir as an item
			wantFileGap:   nil, // Do not record a file gap for the pkg dir
			wantErr:       nil,
		},
		"item and walk error": {
			walkPath:      path.Join(source, pkg, item),
			walkErr:       customWalkError,
			wantItemVisit: nil,             // Do not visit an item with a walk error
			wantFileGap:   nil,             // Do not record a file gap for an item with a walk error
			wantErr:       customWalkError, // Return the walk error
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotItemVisit := false

			visitItem := itemVisitFunc(func(pkg, item string, d fs.DirEntry) error {
				want := test.wantItemVisit
				if want == nil {
					return fmt.Errorf("visit item: unwanted call with pkg %q, item %q",
						pkg, item)
				}
				gotItemVisit = true
				if pkg != want.pkgArg {
					t.Errorf("visit item: want pkg %q, got %q", want.pkgArg, pkg)
				}
				if item != want.itemArg {
					t.Errorf("visit item:, want item %q, got %q", want.itemArg, item)
				}
				return want.errResult
			})

			fsys := duftest.FS{M: test.files}

			pa := NewPkgAnalyst(fsys, source, pkg, visitItem)

			gotErr := pa.VisitPath(test.walkPath, test.walkEntry, test.walkErr)

			if test.wantItemVisit != nil && !gotItemVisit {
				t.Errorf("no call to visit item, wanted: %#v", test.wantItemVisit)
			}

			if !errors.Is(gotErr, test.wantErr) {
				t.Fatalf("error:\nwant %v\ngot  %v", test.wantErr, gotErr)
			}
		})
	}
}
