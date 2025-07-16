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

type testFile struct {
	name string
	mode fs.FileMode
	dest string
	err  errfs.Error
}

func add(tfs *errfs.FS, f *testFile) {
	if f == nil {
		return
	}
	tfs.Add(f.name, f.mode, f.dest, f.err)
}

func TestDirStater(t *testing.T) {
	tests := map[string]struct {
		name      string
		file      *testFile
		destFile  *testFile
		wantState *State
		wantError error
	}{
		"file": {
			name:      "dir/file",
			file:      &testFile{name: "dir/file", mode: 0o644},
			wantState: &State{Mode: 0o644},
		},
		"dir": {
			name:      "dir/dir",
			file:      &testFile{name: "dir/dir", mode: fs.ModeDir | 0o755},
			wantState: &State{Mode: fs.ModeDir | 0o755},
		},
		"link": {
			name: "dir/link",
			file: &testFile{
				name: "dir/link",
				mode: fs.ModeSymlink,
				dest: "../dest-dir/dest-file",
			},
			destFile: &testFile{
				name: "dest-dir/dest-file",
				mode: 0o644,
			},
			wantState: &State{
				Mode:     fs.ModeSymlink,
				Dest:     "../dest-dir/dest-file",
				DestMode: 0o644,
			},
		},
		"file lstat error": {
			name: "dir/file",
			file: &testFile{
				name: "dir/file",
				mode: 0o644,
				err:  errfs.ErrLstat,
			},
			wantError: errfs.ErrLstat,
		},
		"file readlink error": {
			name: "dir/link",
			file: &testFile{
				name: "dir/link",
				mode: fs.ModeSymlink,
				err:  errfs.ErrReadLink,
			},
			wantError: errfs.ErrReadLink,
		},
		"dest lstat error": {
			name: "dir/link",
			file: &testFile{
				name: "dir/link",
				mode: fs.ModeSymlink,
				dest: "../dest-dir/dest-file",
			},
			destFile: &testFile{
				name: "dest-dir/dest-file",
				mode: 0o644,
				err:  errfs.ErrLstat,
			},
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

			if !cmp.Equal(state, test.wantState) {
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
