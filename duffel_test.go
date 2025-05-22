package main

import (
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/files/filestest"
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
		args:           []string{"-source", "../../source", "pkg"},
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
	must := filestest.Must(t)
	for name, test := range dirOptTests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			wd := filepath.Join(tmpDir, test.wd)

			must.MkdirAll(filepath.Join(tmpDir, test.pkgItem), 0o755)
			must.MkdirAll(wd, 0o755)

			t.Chdir(wd)

			err := run(test.args)
			if err != nil {
				t.Fatal("run returned error:", err)
			}

			wantTargetPath := filepath.Join(tmpDir, test.wantTargetPath)
			gotDest := must.Readlink(wantTargetPath)

			if gotDest != test.wantTargetDest {
				t.Errorf("want link dest %q, got %q\n", test.wantTargetDest, gotDest)
			}
		})
	}
}
