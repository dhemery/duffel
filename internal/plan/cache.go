package plan

import (
	"github.com/dhemery/duffel/internal/file"
)

type StateCache struct {
	states map[string]*file.State
	miss   Stater
}

// NewStateCache returns a new, empty state cache that retrieves missing items via miss.
func NewStateCache(miss Stater) *StateCache {
	return &StateCache{
		states: map[string]*file.State{},
		miss:   miss,
	}
}

// State returns the cached state of item.
// If c does not already contain the item
// Get caches and returns the state returned by miss.
func (c *StateCache) State(item string) (*file.State, error) {
	cached, ok := c.states[item]
	if !ok {
		state, err := c.miss.State(item)
		if err != nil {
			return nil, err
		}
		c.states[item] = state
		cached = state
	}
	return cached, nil
}

// SetState sets the cached state of item to state.
func (c *StateCache) SetState(item string, state *file.State) {
	c.states[item] = state
}
