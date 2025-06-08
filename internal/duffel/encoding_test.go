package duffel

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"testing"
)

func TestStateEncodeJSON(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{state: State{}, want: `{}`},
		{state: State{Mode: fs.ModeDir | 0o755}, want: `{"mode":"drwxr-xr-x"}`},
		{state: State{Dest: "my/dest"}, want: `{"dest":"my/dest"}`},
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
