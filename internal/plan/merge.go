package plan

import (
	"fmt"
	"log/slog"
)

type Itemizer interface {
	Itemize(name string) (SourcePath, error)
}

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

func NewMerger(itemizer Itemizer, analyst *Analyst) *merger {
	return &merger{
		itemizer: itemizer,
		analyst:  analyst,
	}
}

type merger struct {
	itemizer Itemizer
	analyst  *Analyst
}

func (m merger) Merge(name string, logger *slog.Logger) error {
	mergeItem, err := m.itemizer.Itemize(name)
	if err != nil {
		return &MergeError{Name: name, Err: err}
	}

	mergeOp := NewMergeOp(mergeItem)
	mergeLogger := logger.WithGroup("merge").With("root", mergeItem)
	_, err = m.analyst.Analyze(mergeLogger, mergeOp)
	return err
}
