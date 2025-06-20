package plan

import (
	"io/fs"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

func TestSpecCache(t *testing.T) {
	name := "myItem"
	missState := &file.State{Mode: fs.ModeSymlink, Dest: "miss/state/dest"}

	miss := func(gotName string) (*file.State, error) {
		if gotName != name {
			t.Errorf("miss: want name %s, got %s", gotName, name)
		}
		return missState, nil
	}
	index := NewSpecCache(miss)

	gotState, err := index.Get(name)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(gotState, missState) {
		t.Errorf("state before set:\nwant %v\n got %v", missState, gotState)
	}

	updatedState := &file.State{Mode: fs.ModeSymlink, Dest: "updates/state/dest"}
	index.Set(name, updatedState)

	gotState, err = index.Get(name)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(gotState, updatedState) {
		t.Errorf("state after set:\nwant %v\n got %v", updatedState, gotState)
	}
}
