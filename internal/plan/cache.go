package plan

import (
	"fmt"

	"github.com/dhemery/duffel/internal/file"
)

// A Spec describes the current and desired states of an item.
type Spec struct {
	Current *file.State `json:"current,omitzero"`
	Desired *file.State `json:"desired,omitzero"`
}

func (s Spec) String() string {
	return fmt.Sprintf("%T{Current:%v,Desired:%v}", s, s.Current, s.Desired)
}

type MissFunc func(item string) (*file.State, error)

// A SpecCache collects Specs by item name.
type SpecCache struct {
	specs map[string]Spec
	miss  MissFunc
}

// NewSpecCache returns a new, empty index that retrieves missing items via miss.
func NewSpecCache(miss MissFunc) *SpecCache {
	return &SpecCache{
		specs: map[string]Spec{},
		miss:  miss,
	}
}

// Get returns the desired state of item.
// If c does not already contain the item
// Get sets the current and desired states
// to the state returned by miss.
func (c *SpecCache) Get(item string) (*file.State, error) {
	spec, ok := c.specs[item]
	if !ok {
		state, err := c.miss(item)
		if err != nil {
			return nil, err
		}
		spec = Spec{Current: state, Desired: state}
		c.specs[item] = spec
	}
	return spec.Desired, nil
}

// Set sets the desired state of item to state.
func (c *SpecCache) Set(item string, state *file.State) {
	spec := c.specs[item]
	spec.Desired = state
	c.specs[item] = spec
}

type Index struct {
	states map[string]*file.State
	miss   Stater
}

// NewIndex returns a new, empty index that retrieves missing items via miss.
func NewIndex(miss Stater) *Index {
	return &Index{
		states: map[string]*file.State{},
		miss:   miss,
	}
}

// State returns the state of item.
// If c does not already contain the item
// Get retrieves and caches the state returned by miss.
func (i *Index) State(item string) (*file.State, error) {
	cached, ok := i.states[item]
	if !ok {
		state, err := i.miss.State(item)
		if err != nil {
			return nil, err
		}
		i.states[item] = state
		cached = state
	}
	return cached, nil
}

// SetState sets the state of item to state.
func (i *Index) SetState(item string, state *file.State) {
	i.states[item] = state
}
