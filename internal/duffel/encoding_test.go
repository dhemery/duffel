package duffel

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"testing"
)

func TestFileStateEncodeJSON(t *testing.T) {
	tests := []struct {
		state FileState
		want  string
	}{
		{
			state: FileState{},
			want:  `{"mode":"----------"}`,
		},
		{
			state: FileState{Mode: fs.ModeDir | 0o755},
			want:  `{"mode":"drwxr-xr-x"}`,
		},
		{
			state: FileState{Mode: fs.ModeSymlink, Dest: "my/dest"},
			want:  `{"mode":"L---------","dest":"my/dest"}`,
		},
		{
			state: FileState{Mode: 0o644}, // Regular file
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
