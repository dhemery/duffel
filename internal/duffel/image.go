package duffel

import (
	"fmt"
	"io/fs"
	"maps"
	"slices"
)

type Image map[string]Status

func (i Image) Create(item string, state *State) {
	i[item] = Status{Desired: state}
}

func (i Image) Status(item string) (Status, bool) {
	s, ok := i[item]
	return s, ok
}

func (i Image) Tasks() []Task {
	var tasks []Task
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(i)) {
		status := i[item]
		if status.Desired == nil {
			continue
		}

		task := Task{Item: item, State: *status.Desired}
		tasks = append(tasks, task)
	}
	return tasks
}

type Status struct {
	Current *State `json:"current,omitzero"`
	Desired *State `json:"desired,omitzero"`
}

func (s Status) String() string {
	return fmt.Sprintf("Status{Current:%v,Desired:%v}", s.Current, s.Desired)
}

type State struct {
	Mode fs.FileMode `json:"mode,omitzero"`
	Dest string      `json:"dest,omitzero"`
}
