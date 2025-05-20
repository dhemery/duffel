package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

type dirOptTest struct {
	pkgItem        string
	wd             string
	args           []string
	wantTargetPath string
	wantTargetDest string
}

var dirOptTests = map[string]dirOptTest{
	"Default source and target": {
		wd:             "home/user/source",
		pkgItem:        "home/user/source/pkg/pkgItem",
		args:           []string{"pkg"},
		wantTargetPath: "home/user/pkgItem",
		wantTargetDest: "source/pkg/pkgItem",
	},
	"Default target, given source": {
		wd:             "home/user/target/wd",
		pkgItem:        "home/user/source/pkg/pkgItem",
		args:           []string{"-source", "../source", "pkg"},
		wantTargetPath: "home/user/target/pkgItem",
		wantTargetDest: "../source/pkg/pkgItem",
	},
	"Default source, given target": {
		wd:             "home/user/source",
		pkgItem:        "home/user/source/pkg/pkgItem",
		args:           []string{"-target", "../target", "pkg"},
		wantTargetPath: "home/user/target/pkgItem",
		wantTargetDest: "../source/pkg/pkgItem",
	},
	"Given source and target": {
		wd:             "home/user/wd",
		pkgItem:        "home/user/source/pkg/pkgItem",
		args:           []string{"-source", "../source", "-target", "../target", "pkg"},
		wantTargetPath: "home/user/target/pkgItem",
		wantTargetDest: "../source/pkg/pkgItem",
	},
}

func TestDirOptions(t *testing.T) {
	for name, test := range dirOptTests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			wd := filepath.Join(tmpDir, test.wd)

			mustMkDir(t, filepath.Join(tmpDir, test.pkgItem))
			mustMkDir(t, wd)

			t.Chdir(wd)

			run(test.args)

			wantTargetPath := filepath.Join(tmpDir, test.wantTargetPath)
			gotDest := mustReadLink(t, wantTargetPath)

			if gotDest != test.wantTargetDest {
				t.Errorf("want link dest %q, got %q\n", test.wantTargetDest, gotDest)
			}
		})
	}
}

func mustMkDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustReadLink(t *testing.T, path string) string {
	t.Helper()
	entry, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}

	gotType := entry.Mode() & fs.ModeType
	if gotType != fs.ModeSymlink {
		t.Fatalf("want a link (file type %O), got file %q is type %O\n", fs.ModeSymlink, path, gotType)
	}

	gotDest, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	return gotDest
}
