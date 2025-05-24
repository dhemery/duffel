package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/files/filestest"
)

// TestMain executes the test binary as the duffel command if
// DUFFEL_TEST_RUN_MAIN is set, and runs the tests otherwise.
func TestMain(m *testing.M) {
	if os.Getenv("DUFFEL_TEST_RUN_MAIN") != "" {
		main()
		os.Exit(0)
	}

	os.Setenv("DUFFEL_TEST_RUN_MAIN", "1") // Set for subprocesses to inherit.
	os.Exit(m.Run())
}

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

// TestDirOptions exercises how run() applies its -source and -target options
// to the actual file system.
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
