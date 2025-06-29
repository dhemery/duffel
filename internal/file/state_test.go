package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/google/go-cmp/cmp"
)

func TestStateLoader(t *testing.T) {
	var (
		anLstatError    = errors.New("error returned from lstat")
		aReadLinkError  = errors.New("error returned from readlink")
		aDestLstatError = errors.New("error returned from dest lstat")
	)

	tests := map[string]struct {
		itemName  string
		files     fs.FS
		wantState *State
		wantError error
	}{
		"file": {
			itemName:  "item",
			files:     duftest.TestFS{"item": duftest.TestFile{Mode: 0o644}},
			wantState: &State{Mode: 0o644},
		},
		"dir": {
			itemName:  "item",
			files:     duftest.TestFS{"item": duftest.TestFile{Mode: fs.ModeDir | 0o755}},
			wantState: &State{Mode: fs.ModeDir | 0o755},
		},
		"link": {
			itemName: "dir1/dir2/item",
			files: duftest.TestFS{
				"dir1/dir2/item": duftest.TestFile{Mode: fs.ModeSymlink, Dest: "../dest"},
				"dir1/dest":      duftest.TestFile{Mode: 0o644},
			},
			wantState: &State{Mode: fs.ModeSymlink, Dest: "../dest", DestMode: 0o644},
		},
		"file lstat error": {
			itemName:  "item",
			files:     duftest.TestFS{"item": duftest.TestFile{LstatErr: anLstatError}},
			wantError: anLstatError,
		},
		"file readlink error": {
			itemName: "item",
			files: duftest.TestFS{
				"item": duftest.TestFile{Mode: fs.ModeSymlink, ReadLinkErr: aReadLinkError},
			},
			wantError: aReadLinkError,
		},
		"dest lstat error": {
			itemName: "dir1/item",
			files: duftest.TestFS{
				"dir1/item": duftest.TestFile{Mode: fs.ModeSymlink, Dest: "../dest"},
				"dest":      duftest.TestFile{LstatErr: aDestLstatError},
			},
			wantError: aDestLstatError,
		},
		"no file": {
			itemName:  "item",
			files:     duftest.TestFS{"item": duftest.TestFile{LstatErr: fs.ErrNotExist}},
			wantState: nil,
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fsys := test.files
			loader := StateLoader{FS: fsys}

			state, err := loader.Load(test.itemName)

			if !errors.Is(err, test.wantError) {
				t.Errorf("error: want %v, got %v", test.wantError, err)
			}

			if !cmp.Equal(state, test.wantState) {
				t.Errorf("state:\nwant %v\n got %v", test.wantState, state)
			}
		})
	}
}

func TestStateEncodeJSON(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{
			state: State{},
			want:  `{"mode":"----------"}`,
		},
		{
			state: State{Mode: fs.ModeDir | 0o755},
			want:  `{"mode":"drwxr-xr-x"}`,
		},
		{
			state: State{Mode: fs.ModeSymlink, Dest: "my/dest"},
			want:  `{"mode":"L---------","dest":"my/dest"}`,
		},
		{
			state: State{Mode: 0o644}, // Regular file
			want:  `{"mode":"-rw-r--r--"}`,
		},
	}

	for _, test := range tests {
		var bb bytes.Buffer
		enc := json.NewEncoder(&bb)

		err := enc.Encode(test.state)
		got := bb.String()

		if err != nil {
			t.Fatalf("%s\n   %q", err, got)
		}

		want := test.want + "\n"
		if got != want {
			t.Errorf("%s\n  want: %q\n  got : %q", test.state, want, got)
		}
	}
}
