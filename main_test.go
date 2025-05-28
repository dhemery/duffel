package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
	wd        string // The working directory in which duffel is run.
	sourceOpt string // The value for the -source option.
	targetOpt string // The value for the -target option.
	wantDest  string // The desired destination for the target link.
}

const (
	pkgName       = "pkg"
	itemName      = "pkgItem"
	defaultSource = "."
	defaultTarget = ".."
)

var dirOptTests = map[string]dirOptTest{
	"Default source and target": {
		wd:        "home/user/wd",
		sourceOpt: "", // home/user/wd
		targetOpt: "", // home/user
		wantDest:  filepath.Join("wd", pkgName, itemName),
	},
	"Default target, given source": {
		wd:        "home/user/target/wd",
		sourceOpt: "../../source", // home/user/source
		targetOpt: "",             // home/user/target
		wantDest:  filepath.Join("../source", pkgName, itemName),
	},
	"Default source, given target": {
		wd:        "home/user/wd",
		sourceOpt: "",          // home/user/source
		targetOpt: "../target", // home/user/target
		wantDest:  filepath.Join("../wd", pkgName, itemName),
	},
	"Given source and target": {
		wd:        "home/user/wd",
		sourceOpt: "../source", // home/user/source
		targetOpt: "../target", // home/user/target
		wantDest:  filepath.Join("../source", pkgName, itemName),
	},
}

// TestDirOptions tests how the duffel command maps the -source and -target options
// to file system entries.
func TestDirOptions(t *testing.T) {
	for name, test := range dirOptTests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			wd := filepath.Join(tmpDir, test.wd)
			sourceDir := filepath.Join(wd, cmp.Or(test.sourceOpt, defaultSource))
			targetDir := filepath.Join(wd, cmp.Or(test.targetOpt, defaultTarget))
			itemPath := filepath.Join(sourceDir, pkgName, itemName)

			// Making itemPath necessarily makes sourceDir
			if err := mkAllDirs(wd, itemPath, targetDir); err != nil {
				t.Error(err)
				return
			}

			args := []string{}
			if test.sourceOpt != "" {
				args = append(args, "-source", test.sourceOpt)
			}
			if test.targetOpt != "" {
				args = append(args, "-target", test.targetOpt)
			}
			args = append(args, pkgName)

			td := testDuffel(wd, args...)

			wantTargetPath := filepath.Join(targetDir, itemName)

			if err := td.Run(); err != nil {
				t.Error(err)
				if info, err := os.Lstat(wantTargetPath); err == nil {
					t.Log(wantTargetPath, fs.FormatFileInfo(info))
				}
				t.Log("stderr:", td.stderr.String())
				return
			}

			gotDest, err := os.Readlink(wantTargetPath)
			if err != nil {
				t.Error(err)
			}

			if gotDest != test.wantDest {
				t.Errorf("want link dest %q, got %q\n", test.wantDest, gotDest)
			}
		})
	}
}

func TestDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "home/user")
	sourceDir := filepath.Join(targetDir, "source")
	pkgItem := filepath.Join(sourceDir, "pkg/pkgItem")

	// Also creates target and source, which are ancestors
	if err := os.MkdirAll(pkgItem, 0o755); err != nil {
	}

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

	output := td.stdout.String()
	targetItemDest, _ := filepath.Rel(targetDir, pkgItem)
	want := fmt.Sprintf("%s --> %s", targetItemPath[1:], targetItemDest)
	if !strings.Contains(output, want) {
		t.Error("output missing:", want)
		t.Log("output was", output)
	}
}

func mkAllDirs(paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}
