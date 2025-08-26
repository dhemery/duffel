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
		return &mergeError{Dir: name, Err: err}
	}

	mergeOp := mergeDir(mergeItem)
	return m.analyst.analyze(mergeOp, logger)
}

// A mergeError is an error that prevents merging the previously items from a directory.
type mergeError struct {
	Dir string `json:"dir"` // The name of the directory being merged.
	Err error  `json:"err"` // The error that prevents merging.
}

func (me *mergeError) Error() string {
	return fmt.Sprintf("cannot merge %q: %s", me.Dir, me.Err)
}

func (me *mergeError) Unwrap() error {
	return me.Err
}
