package analyze_test

import (
	"path"
	"testing"

	. "github.com/dhemery/duffel/internal/analyze"
)

func TestSourcePath(t *testing.T) {
	tests := map[string]struct {
		sp             SourcePath
		wantString     string
		wantPackageDir string
	}{
		"non-empty item field": {
			sp:             NewSourcePath("s1/s2/s3", "pkg", "i1/i2/i3"),
			wantString:     "s1/s2/s3/pkg/i1/i2/i3",
			wantPackageDir: "s1/s2/s3/pkg",
		},
		"empty item field": {
			sp:             NewSourcePath("s1/s2/s3", "pkg", ""),
			wantString:     "s1/s2/s3/pkg",
			wantPackageDir: "s1/s2/s3/pkg",
		},
	}
	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			if got := test.sp.String(); got != test.wantString {
				t.Errorf("String()=%q, want %q",
					got, test.wantString)
			}
			if got := test.sp.PackageDir(); got != test.wantPackageDir {
				t.Errorf("PackageDir()=%q, want %q",
					got, test.wantPackageDir)
			}

			otherItem := "other/item"
			otherItemFullPath := path.Join(test.wantPackageDir, otherItem)

			if got := test.sp.WithItem(otherItem).String(); got != otherItemFullPath {
				t.Errorf("%q.WithItem(%q)=%q, want %q",
					test.wantString, otherItem, got, otherItemFullPath)
			}

			if got := test.sp.WithItemFrom(otherItemFullPath); got.String() != otherItemFullPath {
				t.Errorf("%q.WithItemFrom(%q)=%q, want %q",
					test.wantString, otherItemFullPath, got, otherItem)
			}
		})
	}
}
