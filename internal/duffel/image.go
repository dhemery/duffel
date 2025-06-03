package duffel

import (
	"io/fs"
	"maps"
	"slices"
)

type Image map[string]Status

func (i Image) Create(item string, state State) {
	i[item] = Status{Planned: state}
}

func (i Image) Status(item string) Status {
	return i[item]
}

func (i Image) Tasks() []Task {
	var tasks []Task
	// Must sort tasks in lexical order by item
	for _, item := range slices.Sorted(maps.Keys(i)) {
		status := i[item]
		if !status.Planned.Exists() {
			continue
		}

		task := Task{Item: item, State: status.Planned}
		tasks = append(tasks, task)
	}
	return tasks
}

type Status struct {
	Existing State
	Planned  State
}

func (s Status) WillExist() bool {
	return s.Existing.Exists() || s.Planned.Exists()
}

type State struct {
	Mode fs.FileMode `json:"mode,omitzero"`
	Dest string      `json:"dest,omitzero"`
}

func (s State) Exists() bool {
	return s.Dest != ""
}
