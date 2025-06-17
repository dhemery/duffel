package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/duftest"
)

func TestStateLoader(t *testing.T) {
	const itemName = "item"
	var (
		anLstatError   = errors.New("error returned from lstat")
		aReadLinkError = errors.New("error returned from readlink")
	)

	tests := map[string]struct {
		file      duftest.TestFile
		wantState *State
		wantError error
	}{
		"file": {
			file:      duftest.TestFile{Mode: 0o644},
			wantState: &State{Mode: 0o644},
		},
		"dir": {
			file:      duftest.TestFile{Mode: fs.ModeDir | 0o755},
			wantState: &State{Mode: fs.ModeDir | 0o755},
		},
		"link": {
			file:      duftest.TestFile{Mode: fs.ModeSymlink, Dest: "test/link/dest"},
			wantState: &State{Mode: fs.ModeSymlink, Dest: "test/link/dest"},
		},
		"lstat error": {
			file:      duftest.TestFile{LstatErr: anLstatError},
			wantError: anLstatError,
		},
		"readlink error": {
			file:      duftest.TestFile{Mode: fs.ModeSymlink, ReadLinkErr: aReadLinkError},
			wantError: aReadLinkError,
		},
		"no file": {
			file:      duftest.TestFile{LstatErr: fs.ErrNotExist},
			wantState: nil,
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fsys := duftest.TestFS{itemName: test.file}
			loader := StateLoader{FS: fsys}

			state, err := loader.Load(itemName)

			if !errors.Is(err, test.wantError) {
				t.Errorf("error: want %v, got %v", test.wantError, err)
			}

			if !reflect.DeepEqual(state, test.wantState) {
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
