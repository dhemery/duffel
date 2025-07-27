package analyze

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
	"github.com/google/go-cmp/cmp"
)

type indexCall func(i *index, t *testing.T)

func get(name string, wantState *file.State, wantErr error) indexCall {
	return func(i *index, t *testing.T) {
		t.Helper()
		state, err := i.State(name)
		if !cmp.Equal(state, wantState) {
			t.Errorf("State(%q) state:\n got: %v\nwant: %v",
				name, state, wantState)
		}
		if !errors.Is(err, wantErr) {
			t.Errorf("State(%q) error:\n got: %v\nwant: %v",
				name, err, wantErr)
		}
	}
}

func set(name string, state *file.State) indexCall {
	return func(i *index, t *testing.T) {
		i.SetState(name, state)
	}
}

func newOneTimeStater(s Stater) Stater {
	return oneTimeStater{s: s, calls: map[string]int{}}
}

// oneTimeStater is a Stater that returns an error
// if State is called more than once with the same name.
type oneTimeStater struct {
	s     Stater
	calls map[string]int
}

func (ots oneTimeStater) State(name string) (*file.State, error) {
	called := ots.calls[name]
	called++
	ots.calls[name] = called

	if called > 1 {
		return nil, fmt.Errorf("oneTimeStater.State(%q) called %d times", name, called)
	}
	return ots.s.State(name)
}

func TestIndex(t *testing.T) {
	tests := map[string]struct {
		files     []*errfs.File
		calls     []indexCall
		wantSpecs map[string]Spec
	}{
		"get state of non-existent file": {
			files: []*errfs.File{}, // No files.
			calls: []indexCall{
				// Two get calls...
				get("target/file", nil, nil),
				// The second call checks (via oneTimeStater) that the index
				// has cached the spec and does not call the file stater again.
				get("target/file", nil, nil),
			},
			wantSpecs: map[string]Spec{
				"target/file": {Current: nil, Planned: nil},
			},
		},
		"get state of existing file": {
			files: []*errfs.File{
				errfs.NewFile("target/file", 0o644),
			},
			calls: []indexCall{
				get("target/file", fileState(), nil),
				get("target/file", fileState(), nil),
			},
			wantSpecs: map[string]Spec{
				"target/file": {Current: fileState(), Planned: fileState()},
			},
		},
		"get state of existing dir": {
			files: []*errfs.File{
				errfs.NewDir("target/dir", 0o755),
			},
			calls: []indexCall{
				get("target/dir", dirState(), nil),
				get("target/dir", dirState(), nil),
			},
			wantSpecs: map[string]Spec{
				"target/dir": {Current: dirState(), Planned: dirState()},
			},
		},
		"get state of existing link": {
			files: []*errfs.File{
				errfs.NewLink("target/link", "../some/dest/file"),
				errfs.NewFile("some/dest/file", 0o644),
			},
			calls: []indexCall{
				get("target/link", linkState("../some/dest/file", 0), nil),
				get("target/link", linkState("../some/dest/file", 0), nil),
			},
			wantSpecs: map[string]Spec{
				"target/link": {
					Current: linkState("../some/dest/file", 0),
					Planned: linkState("../some/dest/file", 0),
				},
			},
		},
		"error getting state of existing file": {
			files: []*errfs.File{
				errfs.NewFile("target/file", 0o644, errfs.ErrLstat),
			},
			calls: []indexCall{
				get("target/file", nil, errfs.ErrLstat),
			},
			wantSpecs: map[string]Spec{},
		},
		"set planned state of non-existent file": {
			files: []*errfs.File{},
			calls: []indexCall{
				get("target/file", nil, nil),
				set("target/file", linkState("link/to/source/file", 0)),
			},
			wantSpecs: map[string]Spec{
				"target/file": {
					Current: nil,
					Planned: linkState("link/to/source/file", 0),
				},
			},
		},
		"set planned state of non-existent dir": {
			files: []*errfs.File{},
			calls: []indexCall{
				get("target/dir", nil, nil),
				set("target/dir", linkState("link/to/source/dir", fs.ModeDir)),
			},
			wantSpecs: map[string]Spec{
				"target/dir": {
					Current: nil,
					Planned: linkState("link/to/source/dir", fs.ModeDir),
				},
			},
		},
	}
	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			var logbuf bytes.Buffer
			SetTestLogger(log.NewJSONLogger(slog.LevelInfo, &logbuf), t)

			testFS := errfs.New()
			for _, f := range test.files {
				errfs.Add(testFS, f)
			}
			testStater := newOneTimeStater(file.NewStater(testFS))

			index := NewIndex(testStater)

			for _, call := range test.calls {
				call(index, t)
			}

			specs := maps.Collect(index.All())
			if diff := cmp.Diff(test.wantSpecs, specs); diff != "" {
				t.Errorf("Specs() after calls: %s", diff)
			}

			if t.Failed() || testing.Verbose() {
				t.Log("files:\n", testFS)
				t.Log("log:\n", logbuf.String())
			}
		})
	}
}
