package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dhemery/duffel/internal/testfs"
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
	t *testing.T
	*exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (td *testDuffelData) DumpIfTestFails() {
	if td.t.Failed() {
		td.t.Logf("stdout: %v", td.stdout)
		td.t.Logf("stderr: %v", td.stderr)
	}
}

func testDuffel(t *testing.T, dir string, args ...string) *testDuffelData {
	cmd := exec.Command(os.Args[0], args...)
	td := testDuffelData{t: t, Cmd: cmd}
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

			must := testfs.Must(t)
			must.MkdirAll(wd, 0o755)
			must.MkdirAll(targetDir, 0o755)
			must.MkdirAll(itemPath, 0o755) // Also necessarily makes sourceDir

			args := []string{}
			if test.sourceOpt != "" {
				args = append(args, "-source", test.sourceOpt)
			}
			if test.targetOpt != "" {
				args = append(args, "-target", test.targetOpt)
			}
			args = append(args, pkgName)

			td := testDuffel(t, wd, args...)
			defer td.DumpIfTestFails()

			if err := td.Run(); err != nil {
				t.Fatal(err)
			}

			installedItemPath := filepath.Join(targetDir, itemName)
			gotDest, err := os.Readlink(installedItemPath)
			if err != nil {
				t.Fatal(err)
			}

			if gotDest != test.wantDest {
				t.Errorf("want link dest %q, got %q\n", test.wantDest, gotDest)
			}
		})
	}
}

func TestDryRun(t *testing.T) {
	tmpDir := t.TempDir()
	pkgName := "pkg"
	itemName := "pkgItem"
	targetDir := filepath.Join(tmpDir, "home/user")
	sourceDir := filepath.Join(targetDir, "source")
	pkgPath := filepath.Join(sourceDir, pkgName)
	itemPath := filepath.Join(pkgPath, itemName)

	must := testfs.Must(t)
	// Also creates target and source, which are ancestors
	must.MkdirAll(itemPath, 0o755)

	// default source (.) and target (..)
	td := testDuffel(t, sourceDir, "-n", "pkg")
	defer td.DumpIfTestFails()

	if err := td.Run(); err != nil {
		t.Fatal(err)
	}

	targetItemPath := filepath.Join(targetDir, "pkgItem")
	info, err := os.Stat(targetItemPath)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want error %q, got %v", fs.ErrNotExist, err)
	}
	if err == nil && info != nil {
		t.Error("created target item:", fs.FormatFileInfo(info))
	}

	var plan struct {
		Target string
		Tasks  []map[string]string
	}
	err = json.Unmarshal(td.stdout.Bytes(), &plan)
	if err != nil {
		t.Fatal(err)
	}

	tasks := plan.Tasks
	if len(plan.Tasks) == 0 {
		t.Fatal("no tasks planned")
	}
	task := tasks[0]

	gotAction := task["Action"]
	wantAction := "link"
	if gotAction != wantAction {
		t.Errorf("want action %q, got, %q", wantAction, gotAction)
	}

	gotPath := task["Path"]
	wantPath := itemName
	if gotPath != wantPath {
		t.Errorf("want path %q, got %q", wantPath, gotPath)
	}

	gotDest := task["Dest"]
	wantDest, _ := filepath.Rel(targetDir, itemPath)
	if gotDest != wantDest {
		t.Errorf("want dest %q, got %q", wantDest, gotDest)
	}
}
