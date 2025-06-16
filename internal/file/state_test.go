package file

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"reflect"
	"testing"
	"time"
)

type testFileInfo struct {
	mode          fs.FileMode
	dest          string
	lstatError    error
	readLinkError error
}

func (t testFileInfo) IsDir() bool {
	return t.Mode()&fs.ModeDir != 0
}

func (t testFileInfo) ModTime() time.Time {
	return time.Now()
}

func (t testFileInfo) Mode() fs.FileMode {
	return t.mode
}

func (t testFileInfo) Name() string {
	return ""
}

func (t testFileInfo) Size() int64 {
	return 0
}

func (t testFileInfo) Sys() any {
	return nil
}

type testFS map[string]testFileInfo

func (fsys testFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "testfs.open", Path: name, Err: errors.ErrUnsupported}
}

func (fsys testFS) Lstat(name string) (fs.FileInfo, error) {
	info, ok := fsys[name]
	if !ok {
		return nil, &fs.PathError{Op: "testfs.lstat", Path: name, Err: fs.ErrNotExist}
	}
	return info, info.lstatError
}

func (fsys testFS) ReadLink(name string) (string, error) {
	info, ok := fsys[name]
	if !ok {
		return "", &fs.PathError{Op: "testfs.readlink", Path: name, Err: fs.ErrNotExist}
	}
	if info.mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: "testfs.readlink", Path: name, Err: fs.ErrInvalid}
	}
	return info.dest, info.readLinkError
}

func TestStateLoader(t *testing.T) {
	const itemName = "item"
	var (
		anLstatError   = errors.New("error returned from lstat")
		aReadLinkError = errors.New("error returned from readlink")
	)

	tests := map[string]struct {
		file      testFileInfo
		wantState *State
		wantError error
	}{
		"file": {
			file:      testFileInfo{mode: 0o644},
			wantState: &State{Mode: 0o644},
		},
		"dir": {
			file:      testFileInfo{mode: fs.ModeDir | 0o755},
			wantState: &State{Mode: fs.ModeDir | 0o755},
		},
		"link": {
			file:      testFileInfo{mode: fs.ModeSymlink, dest: "test/link/dest"},
			wantState: &State{Mode: fs.ModeSymlink, Dest: "test/link/dest"},
		},
		"lstat error": {
			file:      testFileInfo{lstatError: anLstatError},
			wantError: anLstatError,
		},
		"readlink error": {
			file: testFileInfo{
				mode:          fs.ModeSymlink,
				readLinkError: aReadLinkError,
			},
			wantError: aReadLinkError,
		},
		"no file": {
			file:      testFileInfo{lstatError: fs.ErrNotExist},
			wantState: nil,
			wantError: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fsys := testFS{itemName: test.file}
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
