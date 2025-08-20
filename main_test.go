package main

import (
	"bytes"
	. "cmp"
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/plan"
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

// TestDirOptions tests how the duffel command maps the -source and -target options
// to file system entries.
func TestDirOptions(t *testing.T) {
	const (
		pkg           = "pkg"
		item          = "item"
		defaultSource = "."
		defaultTarget = ".."
	)

	tests := []struct {
		desc      string // Description of the test.
		wd        string // The working directory in which duffel is run.
		sourceOpt string // The value for the -source option.
		targetOpt string // The value for the -target option.
		wantDest  string // The desired destination for the target link.
	}{
		{
			desc:      "Default source and target",
			wd:        "home/user/wd",
			sourceOpt: "", // home/user/wd
			targetOpt: "", // home/user
			wantDest:  filepath.Join("wd", pkg, item),
		},
		{
			desc:      "Default target, given source",
			wd:        "home/user/target/wd",
			sourceOpt: "../../source", // home/user/source
			targetOpt: "",             // home/user/target
			wantDest:  filepath.Join("../source", pkg, item),
		},
		{
			desc:      "Default source, given target",
			wd:        "home/user/wd",
			sourceOpt: "",          // home/user/wd
			targetOpt: "../target", // home/user/target
			wantDest:  filepath.Join("../wd", pkg, item),
		},
		{
			desc:      "Given source and target",
			wd:        "home/user/wd",
			sourceOpt: "../source", // home/user/source
			targetOpt: "../target", // home/user/target
			wantDest:  filepath.Join("../source", pkg, item),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			root := t.TempDir()
			wd := filepath.Join(root, test.wd)
			absSource := filepath.Join(wd, Or(test.sourceOpt, defaultSource))
			absDuffelFile := filepath.Join(absSource, ".duffel")
			absTarget := filepath.Join(wd, Or(test.targetOpt, defaultTarget))
			absSourcePkgItem := filepath.Join(absSource, pkg, item)

			must := duftest.Must(t)
			must.MkdirAll(wd, 0o755)
			must.MkdirAll(absTarget, 0o755)
			must.MkdirAll(absSourcePkgItem, 0o755) // Also necessarily makes sourceDir
			must.WriteFile(absDuffelFile, []byte(absDuffelFile), 0o644)

			args := []string{}
			if test.sourceOpt != "" {
				args = append(args, "-source", test.sourceOpt)
			}
			if test.targetOpt != "" {
				args = append(args, "-target", test.targetOpt)
			}
			args = append(args, pkg)

			td := testDuffel(t, wd, args...)
			defer td.DumpIfTestFails()

			if err := td.Run(); err != nil {
				t.Fatal(err)
			}

			targetItem := filepath.Join(absTarget, item)
			gotDest, err := os.Readlink(targetItem)
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
	root := t.TempDir()
	pkg := "pkg"
	item := "item"
	absTarget := filepath.Join(root, "home/user")
	absSource := filepath.Join(absTarget, "source")
	absDuffelFile := filepath.Join(absSource, ".duffel")
	absSourcePkg := filepath.Join(absSource, pkg)
	absSourcePkgItem := filepath.Join(absSourcePkg, item)

	must := duftest.Must(t)
	// Also creates target and source, which are ancestors
	must.MkdirAll(absSourcePkgItem, 0o755)
	must.WriteFile(absDuffelFile, []byte(absDuffelFile), 0o644)

	// default source (.) and target (..)
	td := testDuffel(t, absSource, "-n", "pkg")
	defer td.DumpIfTestFails()

	if err := td.Run(); err != nil {
		t.Fatal(err)
	}

	absTargetItem := filepath.Join(absTarget, item)
	info, err := os.Stat(absTargetItem)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("want error %q, got %v", fs.ErrNotExist, err)
	}
	if err == nil && info != nil {
		t.Error("created target item:", fs.FormatFileInfo(info))
	}

	var gotPlan plan.Plan
	if err = json.Unmarshal(td.stdout.Bytes(), &gotPlan); err != nil {
		t.Fatal(err)
	}

	wantDest, _ := filepath.Rel(absTarget, absSourcePkgItem)
	wantOp := plan.Action{Action: "symlink", Dest: wantDest}
	wantTask := plan.Task{wantOp}
	wantTasks := map[string]plan.Task{"item": wantTask}
	wantPlan := plan.Plan{Target: absTarget[1:], Tasks: wantTasks}

	if diff := cmp.Diff(wantPlan, gotPlan); diff != "" {
		t.Error("plan:", diff)
	}
}

type testDuffelData struct {
	t *testing.T
	*exec.Cmd
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func (td *testDuffelData) DumpIfTestFails() {
	td.t.Helper()
	if td.t.Failed() {
		td.t.Logf("stdout: %q", td.stdout.String())
		td.t.Logf("stderr: %q", td.stderr.String())
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
