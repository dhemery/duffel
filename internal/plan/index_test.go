package plan

import (
	"bytes"
	"errors"
	"maps"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/dhemery/duffel/internal/duftest"
	"github.com/dhemery/duffel/internal/file"
	"github.com/dhemery/duffel/internal/log"
)

func TestIndex(t *testing.T) {
	var logbuf bytes.Buffer
	logger := log.Logger(&logbuf, duftest.LogLevel)
	defer duftest.Dump(t, "log", &logbuf)

	targetPath := NewTargetPath("target", "some/item")
	targetName := targetPath.String()

	// Index must call stater only once, and cache the result.
	testStater := oneTimeStater{
		t:        t,
		wantName: targetName,
		state:    file.LinkState("some/dest", file.TypeDir),
		err:      nil,
	}

	index := NewIndex(testStater)

	// First call returns the state from stater.
	state, err := index.State(targetPath, logger)
	ctx := "first index.State()"
	checkState(t, ctx, state, testStater.state)
	checkErr(t, ctx, err, testStater.err)
	// Caches a spec with current = planned = state from stater.
	wantInitialSpecs := map[string]Spec{
		targetName: {Current: testStater.state, Planned: testStater.state},
	}
	checkRecordedSpecs(t, ctx, index, wantInitialSpecs)

	// Second call returns the cached state without calling stater again.
	state, err = index.State(targetPath, logger)
	ctx = "second index.State()"
	checkState(t, ctx, state, testStater.state)
	checkErr(t, ctx, err, testStater.err)
	checkRecordedSpecs(t, ctx, index, wantInitialSpecs)

	// SetState sets the planned state and leaves the current state unchanged.
	newState := file.DirState()
	ctx = "index.SetState()"
	index.SetState(targetPath, newState, logger)
	wantUpdatedSpecs := map[string]Spec{
		targetName: {Current: testStater.state, Planned: newState},
	}
	checkRecordedSpecs(t, ctx, index, wantUpdatedSpecs)

	// Third call returns the planned state set by SetState.
	state, err = index.State(targetPath, logger)
	ctx = "updated index.State()"
	checkState(t, ctx, state, newState)
	checkErr(t, ctx, err, testStater.err)
	checkRecordedSpecs(t, ctx, index, wantUpdatedSpecs)
}

func TestIndexStaterError(t *testing.T) {
	var logbuf bytes.Buffer
	logger := log.Logger(&logbuf, duftest.LogLevel)
	defer duftest.Dump(t, "log", &logbuf)

	targetPath := NewTargetPath("target", "some/item")
	targetName := targetPath.String()

	testStater := oneTimeStater{
		t:        t,
		wantName: targetName,
		err:      errors.New("error from stater"),
	}

	index := NewIndex(testStater)

	_, err := index.State(targetPath, logger)

	ctx := "index.State()"
	checkErr(t, ctx, err, testStater.err)
	wantSpecs := map[string]Spec{} // No specs.
	checkRecordedSpecs(t, ctx, index, wantSpecs)
}

func checkErr(t *testing.T, ctx string, got, want error) {
	t.Helper()
	if diff := cmp.Diff(want, got, cmpopts.EquateErrors()); diff != "" {
		t.Errorf("%s: error:\n%s", ctx, diff)
	}
}

func checkState(t *testing.T, ctx string, got, want file.State) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s: state:\n%s", ctx, diff)
	}
}

func checkRecordedSpecs(t *testing.T, ctx string, got Specs, want map[string]Spec) {
	t.Helper()
	gotMap := maps.Collect(got.All())
	if diff := cmp.Diff(want, gotMap); diff != "" {
		t.Errorf("%s: Specs():\n%s", ctx, diff)
	}
}

// A oneTimeStater is a Stater that returns an error
// if State is called more than once.
type oneTimeStater struct {
	t        *testing.T
	wantName string
	calls    int
	state    file.State
	err      error
}

func (ots oneTimeStater) State(name string) (file.State, error) {
	ots.t.Helper()
	ots.calls++
	if ots.calls > 1 {
		ots.t.Errorf("oneTimeStater.State() called %d times (name: %s)", ots.calls, name)
	}

	if name != ots.wantName {
		ots.t.Errorf("oneTimeStater.State() name arg:\n got %s\nwant: %s", name, ots.wantName)
	}

	return ots.state, ots.err
}
