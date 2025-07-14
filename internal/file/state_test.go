package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
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

type fileDesc struct {
	name string
	mode fs.FileMode
	dest string
	err  errfs.Error
}

func TestDirStater(t *testing.T) {
	tests := map[string]struct {
		staterDir string
		fileName  string
		files     []fileDesc
		wantState *State
		wantError error
	}{
		"file": {
			staterDir: "stater-dir",
			fileName:  "file",
			files: []fileDesc{{
				name: "stater-dir/file",
				mode: 0o644,
			}},
			wantState: &State{Mode: 0o644},
		},
		"dir": {
			staterDir: "stater-dir",
			fileName:  "dir",
			files: []fileDesc{{
				name: "stater-dir/dir",
				mode: fs.ModeDir | 0o755,
			}},
			wantState: &State{Mode: fs.ModeDir | 0o755},
		},
		"link": {
			staterDir: "stater-dir",
			fileName:  "link",
			files: []fileDesc{
				{
					name: "stater-dir/link",
					mode: fs.ModeSymlink,
					dest: "../dest-dir/dest-file",
				},
				{
					name: "dest-dir/dest-file",
					mode: 0o644,
				},
			},
			wantState: &State{
				Mode:     fs.ModeSymlink,
				Dest:     "../dest-dir/dest-file",
				DestMode: 0o644,
			},
		},
		"file lstat error": {
			staterDir: "stater-dir",
			fileName:  "file",
			files: []fileDesc{{
				name: "stater-dir/file",
				mode: 0o644,
				err:  errfs.ErrLstat,
			}},
			wantError: errfs.ErrLstat,
		},
		"file readlink error": {
			staterDir: "stater-dir",
			fileName:  "link",
			files: []fileDesc{{
				name: "stater-dir/link",
				mode: fs.ModeSymlink,
				err:  errfs.ErrReadLink,
			}},
			wantError: errfs.ErrReadLink,
		},
		"dest lstat error": {
			staterDir: "stater-dir",
			fileName:  "link",
			files: []fileDesc{
				{
					name: "stater-dir/link",
					mode: fs.ModeSymlink,
					dest: "../dest-dir/dest-file",
				},
				{
					name: "dest-dir/dest-file",
					mode: 0o644,
					err:  errfs.ErrLstat,
				},
			},
			wantError: errfs.ErrLstat,
		},
		"no file": {
			staterDir: "stater-dir",
			fileName:  "missing-file",
			files:     nil,
			wantState: nil,
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()

			for _, f := range test.files {
				testFS.Add(f.name, f.mode, f.dest, f.err)
			}

			stater := NewStater(testFS, test.staterDir)

			state, err := stater.State(test.fileName)

			if !errors.Is(err, test.wantError) {
				t.Errorf("State(%s) error:\n got %v\nwant %v",
					test.fileName, err, test.wantError)
			}

			if !cmp.Equal(state, test.wantState) {
				t.Errorf("State(%s) state:\n got %v\nwant %v",
					test.fileName, state, test.wantState)
			}
		})
	}
}
