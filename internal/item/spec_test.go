package item

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/dhemery/duffel/internal/file"
)

func TestIndexCaching(t *testing.T) {
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
	item := "myItem"

	index := NewIndex(nil)

	gotSpec, err := index.Get(item)
	if err == nil {
		t.Errorf("want error, got none, spec: %v", gotSpec)
	}

	state := &file.State{Mode: fs.ModeSymlink, Dest: "my/item/dest"}
	wantSpec := Spec{Current: state, Desired: state}

	index.Set(item, wantSpec)

	gotSpec, err = index.Get(item)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(gotSpec, wantSpec) {
		t.Errorf("spec:\nwant %v\n got %v", wantSpec, gotSpec)
	}
}

func TestIndexByItem(t *testing.T) {
	type namedSpec struct {
		name string
		spec Spec
	}

	orderedSpecs := []namedSpec{
		{
			name: "dir1/sub1/dir",
			spec: Spec{Current: &file.State{Mode: fs.ModeDir | 0o755}},
		},
		{
			name: "dir1/sub1/file",
			spec: Spec{Current: &file.State{Mode: 0o644}},
		},
		{
			name: "dir1/sub2/link",
			spec: Spec{Current: &file.State{Mode: fs.ModeSymlink, Dest: "some/link1"}},
		},
		{
			name: "dir2/sub1/link",
			spec: Spec{Current: &file.State{Mode: fs.ModeSymlink, Dest: "some/link2"}},
		},
		{
			name: "dir2/sub2/dir",
			spec: Spec{Current: &file.State{Mode: fs.ModeDir | 0o755}},
		},
		{
			name: "dir2/sub2/file",
			spec: Spec{Current: &file.State{Mode: 0o644}},
		},
	}

	index := NewIndex(nil)

	// Add the ordered specs to the index in random order
	for _, i := range rand.Perm(len(orderedSpecs)) {
		index.Set(orderedSpecs[i].name, orderedSpecs[i].spec)
	}

	gotSpecs := []namedSpec{}

	for n, s := range index.ByItem() {
		gotSpecs = append(gotSpecs, namedSpec{n, s})
	}

	gotLen := len(gotSpecs)
	wantLen := len(orderedSpecs)
	checkLen := min(gotLen, wantLen)
	for i := range checkLen {
		gotSpec := gotSpecs[i]
		wantSpec := orderedSpecs[i]
		if !reflect.DeepEqual(gotSpec, wantSpec) {
			t.Errorf("spec %d:\nwant: %q %v,\n got: %q %v",
				i, wantSpec.name, wantSpec.spec, gotSpec.name, gotSpec.spec)
		}
	}
	if gotLen < wantLen {
		for i, s := range orderedSpecs[checkLen:] {
			t.Errorf("missing spec %d:\n%q %v", i+checkLen, s.name, s.spec)
		}
	}
	if gotLen > wantLen {
		for i, s := range gotSpecs[checkLen:] {
			t.Errorf("got extra spec %d:\n%q %v", i+checkLen, s.name, s.spec)
		}
	}
}
