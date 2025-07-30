package analyze

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
	mergeLogger := logger.WithGroup("merging").With("name", name)
	mergeItem, err := m.itemizer.Itemize(name)
	if err != nil {
		return &MergeError{Name: name, Err: err}
	}

	pkgOp := NewMergeOp(mergeItem, OpInstall)
	_, err = m.analyst.Analyze(mergeLogger, pkgOp)
	return err
}
