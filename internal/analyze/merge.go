package analyze

import (
	"fmt"
	"log/slog"
)

type Itemizer interface {
	Itemize(name string) (PackageItem, error)
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

func NewMerger(itemizer Itemizer, analyst *Analyst, logger *slog.Logger) *merger {
	return &merger{
		itemizer: itemizer,
		analyst:  analyst,
		log:      logger,
	}
}

type merger struct {
	itemizer Itemizer
	analyst  *Analyst
	log      *slog.Logger
}

func (m merger) Merge(dir, target string) error {
	mergeItem, err := m.itemizer.Itemize(dir)
	if err != nil {
		return &MergeError{Name: dir, Err: err}
	}

	m.log.Info("merge", "foreign-item", mergeItem)
	pkgOp := NewMergeOp(mergeItem, OpInstall)
	_, err = m.analyst.Analyze(pkgOp)
	return err
}
