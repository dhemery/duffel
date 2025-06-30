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
			t.Errorf("%s\n got: %q\nwant: %q", test.state, got, want)
		}
	}
}

func TestDirStater(t *testing.T) {
	var (
		anLstatError    = errors.New("error returned from lstat")
		aReadLinkError  = errors.New("error returned from readlink")
		aDestLstatError = errors.New("error returned from dest lstat")
	)

	tests := map[string]struct {
		staterDir string
		fileName  string
		files     fs.FS
		wantState *State
		wantError error
	}{
		"file": {
			staterDir: "stater-dir",
			fileName:  "file",
			files: duftest.TestFS{
				"stater-dir/file": duftest.TestFile{Mode: 0o644},
			},
			wantState: &State{Mode: 0o644},
		},
		"dir": {
			staterDir: "stater-dir",
			fileName:  "dir",
			files: duftest.TestFS{
				"stater-dir/dir": duftest.TestFile{Mode: fs.ModeDir | 0o755},
			},
			wantState: &State{Mode: fs.ModeDir | 0o755},
		},
		"link": {
			staterDir: "stater-dir",
			fileName:  "link",
			files: duftest.TestFS{
				"stater-dir/link": duftest.TestFile{
					Mode: fs.ModeSymlink,
					Dest: "../dest-dir/dest-file",
				},
				"dest-dir/dest-file": duftest.TestFile{Mode: 0o644},
			},
			wantState: &State{Mode: fs.ModeSymlink, Dest: "../dest-dir/dest-file", DestMode: 0o644},
		},
		"file lstat error": {
			staterDir: "stater-dir",
			fileName:  "file",
			files: duftest.TestFS{
				"stater-dir/file": duftest.TestFile{LstatErr: anLstatError},
			},
			wantError: anLstatError,
		},
		"file readlink error": {
			staterDir: "stater-dir",
			fileName:  "link",
			files: duftest.TestFS{
				"stater-dir/link": duftest.TestFile{
					Mode:        fs.ModeSymlink,
					ReadLinkErr: aReadLinkError,
				},
			},
			wantError: aReadLinkError,
		},
		"dest lstat error": {
			staterDir: "stater-dir",
			fileName:  "link",
			files: duftest.TestFS{
				"stater-dir/link": duftest.TestFile{
					Mode: fs.ModeSymlink,
					Dest: "../dest-dir/dest-file",
				},
				"dest-dir/dest-file": duftest.TestFile{LstatErr: aDestLstatError},
			},
			wantError: aDestLstatError,
		},
		"no file": {
			staterDir: "stater-dir",
			fileName:  "missing-file",
			files:     duftest.TestFS{},
			wantState: nil,
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			stater := DirStater{FS: test.files, Dir: test.staterDir}

			state, err := stater.State(test.fileName)

			if !errors.Is(err, test.wantError) {
				t.Errorf("State(%s) error: got %v, want %v", test.fileName, err, test.wantError)
			}

			if !cmp.Equal(state, test.wantState) {
				t.Errorf("State(%s) state:\n got %v\nwant %v", test.fileName, state, test.wantState)
			}
		})
	}
}
