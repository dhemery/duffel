package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

type dirOptTest struct {
	dirs           []string
	wd             string
	sourceOpt      string
	targetOpt      string
	pkgs           []string
	wantTargetPath string
	wantTargetDest string
}

var dirOptTests = map[string]dirOptTest{
	"empty source and target options": {
		dirs:           []string{"root/home/user/source/pkg/pkgItem"},
		wd:             "root/home/user/source",
		sourceOpt:      "", // Defaults to .: root/home/user/source
		targetOpt:      "", // Defaults to ..: root/home/user
		pkgs:           []string{"pkg"},
		wantTargetPath: "root/home/user/pkgItem",
		wantTargetDest: "source/pkg/pkgItem",
	},
}

func TestDirOptions(t *testing.T) {
	for name, test := range dirOptTests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			for _, dir := range test.dirs {
				err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755)
				if err != nil {
					t.Fatal("setup: making dir:", err)
				}
			}

			wd := filepath.Join(tmpDir, test.wd)
			t.Chdir(wd)

			run(test.pkgs)

			wantTargetPath := filepath.Join(tmpDir, test.wantTargetPath)
			installedEntry, err := os.Lstat(wantTargetPath)
			if err != nil {
				t.Fatal("installed entry:", err)
			}

			wantType := fs.ModeSymlink
			gotType := installedEntry.Mode() & fs.ModeType
			if gotType != wantType {
				t.Fatalf("got installed file type %O, want %O", gotType, wantType)
			}

			gotDest, err := os.Readlink(wantTargetPath)
			if err != nil {
				t.Fatal("installed link dest:", err)
			}

			if gotDest != test.wantTargetDest {
				t.Errorf("want link dest %q, got %q", test.wantTargetDest, gotDest)
			}
		})
	}
}
