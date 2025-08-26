package plan

import (
	"path"
	"testing"
)

func TestSourcePath(t *testing.T) {
	tests := map[string]struct {
		sourcePath     SourcePath
		wantString     string
		wantPackageDir string
	}{
		"item path": {
			sourcePath:     newSourcePath("s1/s2/s3", "pkg", "i1/i2/i3"),
			wantString:     "s1/s2/s3/pkg/i1/i2/i3",
			wantPackageDir: "s1/s2/s3/pkg",
		},
		"package path": {
			sourcePath:     newSourcePath("s1/s2/s3", "pkg", ""),
			wantString:     "s1/s2/s3/pkg",
			wantPackageDir: "s1/s2/s3/pkg",
		},
	}
	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			if got := test.sourcePath.String(); got != test.wantString {
				t.Errorf("String()=%q, want %q",
					got, test.wantString)
			}
			if got := test.sourcePath.PackageDir(); got != test.wantPackageDir {
				t.Errorf("PackageDir()=%q, want %q",
					got, test.wantPackageDir)
			}

			otherItem := "other/item"
			otherItemFullPath := path.Join(test.wantPackageDir, otherItem)

			if got := test.sourcePath.withItem(otherItem).String(); got != otherItemFullPath {
				t.Errorf("%q.WithItem(%q)=%q, want %q",
					test.wantString, otherItem, got, otherItemFullPath)
			}

			if got := test.sourcePath.withItemFrom(otherItemFullPath); got.String() != otherItemFullPath {
				t.Errorf("%q.WithItemFrom(%q)=%q, want %q",
					test.wantString, otherItemFullPath, got, otherItem)
			}
		})
	}
}

func TestTargetPath(t *testing.T) {
	targetPath := newTargetPath("my/target", "my/item")

	if got, want := targetPath.String(), "my/target/my/item"; got != want {
		t.Errorf("String()=%q, want %q", got, want)
	}

	full := "path/to/other/file"
	rel := "../../../path/to/other/file"

	got := targetPath.PathTo(full)
	if got != rel {
		t.Errorf("PathTo()=%q, want %q", got, rel)
	}

	if got := targetPath.Resolve(rel); got != full {
		t.Errorf("Resolve()=%q, want %q", got, full)
	}
}
