package plan

import (
	"io/fs"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

func TestIndexAccess(t *testing.T) {
	name := "myItem"
	missState := &file.State{Mode: fs.ModeSymlink, Dest: "miss/state/dest"}

	miss := func(gotName string) (*file.State, error) {
		if gotName != name {
			t.Errorf("miss: want name %s, got %s", gotName, name)
		}
		return missState, nil
	}
	index := NewIndex(miss)

	gotState, err := index.Desired(name)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(gotState, missState) {
		t.Errorf("state before set desired:\nwant %v\n got %v", missState, gotState)
	}

	updatedState := &file.State{Mode: fs.ModeSymlink, Dest: "updates/state/dest"}
	index.SetDesired(name, updatedState)

	gotState, err = index.Desired(name)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(gotState, updatedState) {
		t.Errorf("state after set desired:\nwant %v\n got %v", updatedState, gotState)
	}
}
