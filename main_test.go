package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
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

func TestInstallDirErrors(t *testing.T) {
	const (
		doesNotExist    = 0
		permReadable    = 0o755
		permUnreadable  = 0o222
		permUnwriteable = 0o533
	)
	type entry struct {
		path string
		mode fs.FileMode
	}

	tests := map[string]struct {
		sourceMode  fs.FileMode
		targetMode  fs.FileMode
		packageMode fs.FileMode
		wantError   error
	}{
		"package dir missing": {
			sourceMode:  permReadable,
			packageMode: doesNotExist,
			targetMode:  permReadable,
			wantError:   fs.ErrNotExist,
		},
		"package dir unreadable": {
			sourceMode:  permReadable,
			packageMode: permUnreadable,
			targetMode:  permReadable,
			wantError:   fs.ErrPermission,
		},
		"source dir missing": {
			sourceMode:  doesNotExist,
			packageMode: doesNotExist, // Creating package would require source to exist
			targetMode:  permReadable,
			wantError:   fs.ErrNotExist,
		},
		"source dir unreadable": {
			sourceMode:  permUnreadable,
			packageMode: permReadable,
			targetMode:  permReadable,
			wantError:   fs.ErrPermission,
		},
		"target dir missing": {
			sourceMode:  permReadable,
			packageMode: permReadable,
			targetMode:  doesNotExist,
			wantError:   fs.ErrNotExist,
		},
		"target dir unwriteable": {
			sourceMode:  permReadable,
			packageMode: permReadable,
			targetMode:  permUnwriteable,
			wantError:   fs.ErrPermission,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			wd := filepath.Join(tmpDir, "home/user")
			sourcePath := filepath.Join(wd, "source")
			packagePath := filepath.Join(sourcePath, "pkg")
			targetPath := filepath.Join(wd, "target")
			if test.packageMode != doesNotExist {
				itemPath := path.Join(packagePath, "item")
				os.MkdirAll(itemPath, 0o755)
				os.Chmod(packagePath, test.packageMode)
				defer os.Chmod(packagePath, 0o755)
			}
			if test.sourceMode != doesNotExist {
				os.MkdirAll(sourcePath, 0o755)
				os.Chmod(sourcePath, test.sourceMode)
				defer os.Chmod(sourcePath, 0o755)
			}
			if test.targetMode != doesNotExist {
				os.MkdirAll(targetPath, 0o755)
				os.Chmod(targetPath, test.targetMode)
				defer os.Chmod(targetPath, 0o755)
			}

			cmd := testDuffel(wd, "-source", "source", "-target", "target", "pkg")

			if err := cmd.Run(); err == nil {
				t.Error("want error, but duffel succeeded")
				t.Log("package:", stat(packagePath))
				t.Log("source:", stat(sourcePath))
				t.Log("target:", stat(targetPath))
				t.Log("stderr:", cmd.stderr.String())
				t.Log("stdout:", cmd.stdout.String())
			}
		})
	}
}

func stat(path string) string {
	info, err := os.Lstat(path)
	if err == nil {
		return fs.FormatFileInfo(info)
	}
	return err.Error()
}

func mkAllDirs(paths ...string) error {
	for _, p := range paths {
		if err := os.MkdirAll(p, 0o755); err != nil {
			return err
		}
	}
	return nil
}
