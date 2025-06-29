package plan

import (
	"io/fs"
	"testing"

	"github.com/dhemery/duffel/internal/file"
	"github.com/google/go-cmp/cmp"
)

func TestSpecCache(t *testing.T) {
	name := "myItem"
	missState := &file.State{Mode: fs.ModeSymlink, Dest: "miss/state/dest"}

	miss := func(gotName string) (*file.State, error) {
		if gotName != name {
			t.Errorf("miss: got name %s, want %s", name, gotName)
		}
		return missState, nil
	}
	index := NewSpecCache(miss)

	gotState, err := index.Get(name)
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(gotState, missState) {
		t.Errorf("state before set:\n got %v\nwant %v", gotState, missState)
	}

	updatedState := &file.State{Mode: fs.ModeSymlink, Dest: "updates/state/dest"}
	index.Set(name, updatedState)

	gotState, err = index.Get(name)
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(gotState, updatedState) {
		t.Errorf("state after set:\n got %v\nwant %v", gotState, updatedState)
	}
}
