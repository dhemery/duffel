package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
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

type testDuffelData struct {
	*exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func testDuffel(dir string, args ...string) *testDuffelData {
	cmd := exec.Command(os.Args[0], args...)
	td := testDuffelData{Cmd: cmd}
	cmd.Dir = dir
	cmd.Stdout = &td.stdout
	cmd.Stderr = &td.stderr
	return &td
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

// TestDirOptions tests how the duffel command maps the -source and -target options
// to file system entries.
func TestDirOptions(t *testing.T) {
	must := filestest.Must(t)
	for name, test := range dirOptTests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pkgItem := filepath.Join(tmpDir, test.pkgItem)
			wd := filepath.Join(tmpDir, test.wd)

			must.MkdirAll(pkgItem, 0o755)
			must.MkdirAll(wd, 0o755)

			td := testDuffel(wd, test.args...)

			if err := td.Run(); err != nil {
				t.Error(err)
				return
			}

			wantTargetPath := filepath.Join(tmpDir, test.wantTargetPath)
			gotDest, err := os.Readlink(wantTargetPath)
			if err != nil {
				t.Error(err)
			}

			if gotDest != test.wantTargetDest {
				t.Errorf("want link dest %q, got %q\n", test.wantTargetDest, gotDest)
			}
		})
	}
}

func TestDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "home/user")
	sourceDir := filepath.Join(targetDir, "source")
	pkgItem := filepath.Join(sourceDir, "pkg/pkgItem")

	must := filestest.Must(t)
	must.MkdirAll(pkgItem, 0o755) // Also creates target and source, which are ancestors

	// default source (.) and target (..)
	td := testDuffel(sourceDir, "-n", "pkg")

	if err := td.Run(); err != nil {
		t.Error(err)
		return
	}

	targetItemPath := filepath.Join(targetDir, "pkgItem")
	info, err := os.Stat(targetItemPath)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want error %q, got %v", fs.ErrNotExist, err)
	}
	if err == nil && info != nil {
		t.Error("created target item:", fs.FormatFileInfo(info))
	}
}
