package item

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

func TestIndex(t *testing.T) {
	aMissError := errors.New("error returned from miss")

	tests := map[string]struct {
		callCount int
		missState *file.State
		missErr   error
	}{
		"miss success": {
			callCount: 7, // All calls must succeed, but only 1 call to miss
			missState: &file.State{Mode: fs.ModeSymlink, Dest: "miss/success"},
			missErr:   nil,
		},
		"miss error": {
			callCount: 1,
			missState: nil,
			missErr:   aMissError,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			const name = "item/name"

			gotMiss := false
			miss := func(gotName string) (*file.State, error) {
				if gotMiss {
					t.Errorf("miss: extra call with name %s", name)
				}
				gotMiss = true
				if gotName != name {
					t.Errorf("miss: want name %s, got %s", gotName, name)
				}
				return test.missState, test.missErr
			}

			index := NewIndex(miss)

			for i := range test.callCount {
				gotState, err := index.Desired(name)
				if err != test.missErr {
					t.Errorf("call %d error: want%v, got %v", i+1, test.missErr, err)
				}

				if !reflect.DeepEqual(gotState, test.missState) {
					t.Errorf("call %d state:\nwant: %v\n got: %v", i+1, test.missState, gotState)
				}

			}

			if !gotMiss {
				t.Errorf("miss not called")
			}
		})
	}
}

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

func TestIndexByItem(t *testing.T) {
	type itemSpec struct {
		name string
		spec Spec
	}

	newSpec := func(state *file.State) Spec {
		return Spec{Current: state, Desired: state}
	}

	orderedItems := []itemSpec{
		{
			name: "dir1/sub1/dir",
			spec: newSpec(&file.State{Mode: fs.ModeDir | 0o755}),
		},
		{
			name: "dir1/sub1/file",
			spec: newSpec(&file.State{Mode: 0o644}),
		},
		{
			name: "dir1/sub2/link",
			spec: newSpec(&file.State{Mode: fs.ModeSymlink, Dest: "some/link1"}),
		},
		{
			name: "dir2/sub1/link",
			spec: newSpec(&file.State{Mode: fs.ModeSymlink, Dest: "some/link2"}),
		},
		{
			name: "dir2/sub2/dir",
			spec: newSpec(&file.State{Mode: fs.ModeDir | 0o755}),
		},
		{
			name: "dir2/sub2/file",
			spec: newSpec(&file.State{Mode: 0o644}),
		},
	}

	miss := func(name string) (*file.State, error) {
		for _, item := range orderedItems {
			if item.name == name {
				return item.spec.Current, nil
			}
		}
		return nil, fmt.Errorf("miss error: unknown name %q", name)
	}

	index := NewIndex(miss)

	// Miss the ordered items into the index in random order
	for _, i := range rand.Perm(len(orderedItems)) {
		index.Desired(orderedItems[i].name)
	}

	gotItems := []itemSpec{}

	for n, s := range index.ByItem() {
		gotItems = append(gotItems, itemSpec{n, s})
	}

	gotLen := len(gotItems)
	wantLen := len(orderedItems)
	checkLen := min(gotLen, wantLen)

	for i := range checkLen {
		wantItem := orderedItems[i]
		gotItem := gotItems[i]
		if !reflect.DeepEqual(gotItem, wantItem) {
			t.Errorf("spec %d:\nwant: %q %v,\n got: %q %v",
				i, wantItem.name, wantItem.spec, gotItem.name, gotItem.spec)
		}
	}
	if gotLen < wantLen {
		for i, s := range orderedItems[checkLen:] {
			t.Errorf("missing spec %d:\n%q %v", i+checkLen, s.name, s.spec)
		}
	}
	if gotLen > wantLen {
		for i, s := range gotItems[checkLen:] {
			t.Errorf("got extra spec %d:\n%q %v", i+checkLen, s.name, s.spec)
		}
	}
}
