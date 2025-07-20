package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"math/rand/v2"
	"testing"

	"github.com/dhemery/duffel/internal/errfs"
	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
)

type indexFunc func(i *index, t *testing.T)

func get(name string, wantState *file.State, wantErr error) indexFunc {
	return func(i *index, t *testing.T) {
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

func set(name string, state *file.State) indexFunc {
	return func(i *index, t *testing.T) {
		i.SetState(name, state)
	}
}

type staterFunc func(string) (*file.State, error)

func (f staterFunc) State(name string) (*file.State, error) {
	return f(name)
}

// oneTimeStater returns a Stater that returns an error
// on every call after the first.
func oneTimeStater(s Stater) Stater {
	calls := map[string]int{}
	return staterFunc(func(name string) (*file.State, error) {
		called := calls[name]
		called++
		calls[name] = called

		if called > 1 {
			return nil, fmt.Errorf("oneTimeStater.State(%q) called %d times", name, called)
		}
		return s.State(name)
	})
}

func TestIndex(t *testing.T) {
	tests := map[string]struct {
		files     []testFile
		calls     []indexFunc
		wantSpecs map[string]Spec
	}{
		"get current state of non-existent file": {
			files: []testFile{},
			calls: []indexFunc{
				get("target/no/such/file", nil, nil),
			},
			wantSpecs: map[string]Spec{
				"target/no/such/file": {Current: nil, Planned: nil},
			},
		},
		"set planned state of non-existent dir": {
			files: []testFile{},
			calls: []indexFunc{
				get("target/no/such/dir", nil, nil),
				set("target/no/such/dir", linkState("link/to/source/dir", fs.ModeDir)),
			},
			wantSpecs: map[string]Spec{
				"target/no/such/dir": {
					Current: nil,
					Planned: linkState("link/to/source/dir", fs.ModeDir),
				},
			},
		},
	}
	for desc, test := range tests {
		t.Run(desc, func(t *testing.T) {
			testFS := errfs.New()
			for _, f := range test.files {
				testFS.Add(f.name, f.mode, f.dest)
			}
			testStater := oneTimeStater(file.Stater{FS: testFS})

			index := NewIndex(testStater)

			for _, call := range test.calls {
				call(index, t)
			}

			specs := maps.Collect(index.Specs())
			specsDiff := cmp.Diff(test.wantSpecs, specs)
			if specsDiff != "" {
				t.Errorf("Specs() after calls: %s", specsDiff)
			}
		})
	}
}

func TestStateCache(t *testing.T) {
	missState := &file.State{Mode: fs.ModeSymlink, Dest: "miss/state/dest"}
	name := "myItem"

	miss := staterFunc(func(gotName string) (*file.State, error) {
		if gotName != name {
			t.Errorf("miss: got name %s, want %s", name, gotName)
		}
		return missState, nil
	})

	index := NewIndex(miss)

	gotState, err := index.State(name)
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(gotState, missState) {
		t.Errorf("state before set:\n got %v\nwant %v", gotState, missState)
	}

	updatedState := &file.State{Mode: fs.ModeSymlink, Dest: "updated/state/dest"}
	index.SetState(name, updatedState)

	gotState, err = index.State(name)
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(gotState, updatedState) {
		t.Errorf("state after set:\n got %v\nwant %v", gotState, updatedState)
	}
}

type itemState struct {
	Item  string
	State *file.State
}

func (s itemState) String() string {
	return fmt.Sprintf("%s: %s", s.Item, s.State)
}

func TestIndexAll(t *testing.T) {
	// Some items and their states, sorted by item
	want := []itemState{
		{Item: "a/b/c/dir", State: &file.State{Mode: fs.ModeDir | 0o755}},
		{Item: "a/b/c/file", State: &file.State{Mode: 0o644}},
		{Item: "a/b/file", State: &file.State{Mode: 0o644}},
		{Item: "a/symlink", State: &file.State{Mode: fs.ModeSymlink, Dest: "a/symlink/dest", DestMode: 0o644}},
	}
	wantLen := len(want)

	cache := NewIndex(nil)

	// Add the items in a random order
	for _, i := range rand.Perm(wantLen) {
		s := want[i]
		cache.SetState(s.Item, s.State)
	}

	got := []itemState{}
	for item, state := range cache.All() {
		got = append(got, itemState{Item: item, State: state})
	}

	gotLen := len(got)

	for i := range min(gotLen, wantLen) {
		if !cmp.Equal(got[i], want[i]) {
			t.Errorf("item %d\n got %s\nwant %s", i, got[i], want[i])
		}
	}

	if gotLen < wantLen {
		for i := gotLen; i < wantLen; i++ {
			t.Errorf("missing item %d: %s", i, want[i])
		}
	}
	if gotLen > wantLen {
		for i := wantLen; i < gotLen; i++ {
			t.Errorf("extra item %d: %s", i, got[i])
		}
	}
}
