package duffel

import (
	"bytes"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/dhemery/duffel/internal/testfs"
)

func TestInstallFreshTargetOnePackage(t *testing.T) {
	const (
		pkgName = "pkg"
		source  = "home/user/source"
		target  = "home/user/target"
	)
	items := []struct {
		source   string
		target   string
		wantDest string
		info     *fstest.MapFile
	}{
		{
			source:   "home/user/source/pkg/dirItem",
			target:   "home/user/target/dirItem",
			wantDest: "../source/pkg/dirItem",
			info:     testfs.DirEntry(0o755),
		},
		{
			source:   "home/user/source/pkg/fileItem",
			target:   "home/user/target/fileItem",
			wantDest: "../source/pkg/fileItem",
			info:     testfs.FileEntry("ignored content", 0o644),
		},
		{
			source:   "home/user/source/pkg/linkItem",
			target:   "home/user/target/linkItem",
			wantDest: "../source/pkg/linkItem",
			info:     testfs.LinkEntry("ignored/link/dest"),
		},
	}

	tfs := testfs.New()
	tfs.MapFS[target] = testfs.DirEntry(0o755)
	for _, item := range items {
		tfs.MapFS[item.source] = item.info
	}

	req := &Request{
		FS:     tfs,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Source: source,
		Target: target,
		Pkgs:   []string{pkgName},
	}

	err := Install(req)
	if err != nil {
		t.Error(err)
	}

	for _, item := range items {
		installed, ok := tfs.MapFS[item.target]
		if !ok {
			t.Errorf("%q not installed:", item.target)
			continue
		}
		wantMode := fs.ModeSymlink
		if installed.Mode != wantMode {
			t.Errorf("%q want mode %s, got %s", item.target, wantMode, installed.Mode)
		}
		gotDest := string(installed.Data)
		if gotDest != item.wantDest {
			t.Errorf("%q want dest %s, got %s", item.target, item.wantDest, gotDest)
		}
	}
}
