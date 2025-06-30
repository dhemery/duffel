package plan

import (
	"io/fs"
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

	cache := NewStateCache(miss)

	gotState, err := cache.State(name)
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(gotState, missState) {
		t.Errorf("state before set:\n got %v\nwant %v", gotState, missState)
	}

	updatedState := &file.State{Mode: fs.ModeSymlink, Dest: "updated/state/dest"}
	cache.SetState(name, updatedState)

	gotState, err = cache.State(name)
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(gotState, updatedState) {
		t.Errorf("state after set:\n got %v\nwant %v", gotState, updatedState)
	}
}
