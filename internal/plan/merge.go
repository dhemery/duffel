package plan

import (
	"fmt"
	"log/slog"
)

func newMerger(itemizer itemizer, analyst *analyzer) *merger {
	return &merger{
		itemizer: itemizer,
		analyst:  analyst,
	}
}

type merger struct {
	itemizer itemizer
	analyst  *analyzer
}

func (m merger) merge(name string, logger *slog.Logger) error {
	mergeItem, err := m.itemizer.itemize(name)
	if err != nil {
		return &MergeError{Name: name, Err: err}
	}

	mergeOp := mergeDir(mergeItem)
	return m.analyst.analyze(mergeOp, logger)
}

type MergeError struct {
	Name string `json:"name"`
	Err  error  `json:"err"`
}

func (me *MergeError) Error() string {
	return fmt.Sprintf("cannot merge %q: %s", me.Name, me.Err)
}

func (me *MergeError) Unwrap() error {
	return me.Err
}
