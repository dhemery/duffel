package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
)

func add(tfs *errfs.FS, f *errfs.File) {
	if f == nil {
		return
	}
	errfs.Add(tfs, f)
}

func TestStater(t *testing.T) {
	tests := map[string]struct {
		name      string
		file      *errfs.File
		destFile  *errfs.File
		wantState *State
		wantError error
	}{
		"file": {
			name:      "dir/file",
			file:      errfs.NewFile("dir/file", 0o644),
			wantState: &State{Type: 0},
		},
		"dir": {
			name:      "dir/dir",
			file:      errfs.NewDir("dir/dir", 0o755),
			wantState: &State{Type: fs.ModeDir},
		},
		"link": {
			name:     "dir/link",
			file:     errfs.NewLink("dir/link", "../dest-dir/dest-file"),
			destFile: errfs.NewFile("dest-dir/dest-file", 0o644),
			wantState: &State{
				Type:     fs.ModeSymlink,
				Dest:     "../dest-dir/dest-file",
				DestType: 0,
			},
		},
		"file lstat error": {
			name:      "dir/file",
			file:      errfs.NewFile("dir/file", 0o644, errfs.ErrLstat),
			wantError: errfs.ErrLstat,
		},
		"file readlink error": {
			name:      "dir/link",
			file:      errfs.NewLink("dir/link", "bad/dest", errfs.ErrReadLink),
			wantError: errfs.ErrReadLink,
		},
		"dest lstat error": {
			name:      "dir/link",
			file:      errfs.NewLink("dir/link", "../dest-dir/dest-file"),
			destFile:  errfs.NewFile("dest-dir/dest-file", 0o644, errfs.ErrLstat),
			wantError: errfs.ErrLstat,
		},
		"no file": {
			name:      "missing/file",
			file:      nil,
			wantState: nil,
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			testFS := errfs.New()
			add(testFS, test.file)
			add(testFS, test.destFile)

			stater := Stater{FS: testFS}

			state, err := stater.State(test.name)

			if !errors.Is(err, test.wantError) {
				t.Errorf("State(%s) error:\n got %v\nwant %v",
					test.name, err, test.wantError)
			}

			if !state.Equal(test.wantState) {
				t.Errorf("State(%s) state:\n got %v\nwant %v",
					test.name, state, test.wantState)
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
			state: State{Type: fs.ModeDir | 0o755},
			want:  `{"mode":"drwxr-xr-x"}`,
		},
		{
			state: State{Type: fs.ModeSymlink, Dest: "my/dest"},
			want:  `{"mode":"L---------","dest":"my/dest"}`,
		},
		{
			state: State{Type: 0o644}, // Regular file
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
