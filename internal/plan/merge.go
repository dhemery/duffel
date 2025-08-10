package plan

import (
	"fmt"
	"log/slog"
)

type MergeError struct {
	Name string
	Err  error
}

func (me *MergeError) Error() string {
	return fmt.Sprintf("merge %q: %s", me.Name, me.Err)
}

func (me *MergeError) Unwrap() error {
	return me.Err
}

func NewMerger(itemizer itemizer, analyst *analyzer) *merger {
	return &merger{
		itemizer: itemizer,
		analyst:  analyst,
	}
}

type merger struct {
	itemizer itemizer
	analyst  *analyzer
}

func (m merger) Merge(name string, logger *slog.Logger) error {
	mergeItem, err := m.itemizer.Itemize(name)
	if err != nil {
		return &MergeError{Name: name, Err: err}
	}

	mergeOp := NewMergeOp(mergeItem)
	mergeLogger := logger.WithGroup("merge").With("root", mergeItem)
	return m.analyst.Analyze(mergeOp, mergeLogger)
}
