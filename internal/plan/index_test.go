package plan

import (
	"fmt"
	"io/fs"
	"math/rand/v2"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
)

type staterFunc func(name string) (*file.State, error)

func (f staterFunc) State(name string) (*file.State, error) {
	return f(name)
}

func TestStateCache(t *testing.T) {
	missState := &file.State{Mode: fs.ModeSymlink, Dest: "miss/state/dest"}
	name := "myItem"

	miss := staterFunc(func(gotName string) (*file.State, error) {
		if gotName != name {
			t.Errorf("miss: got name %s, want %s", name, gotName)
		}
		return missState, nil
	},
	)

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
