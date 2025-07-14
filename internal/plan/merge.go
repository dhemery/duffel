package plan

type Finder interface {
	FindPkg(name string) (PkgOp, error)
}

func NewMerger(finder Finder, analyzer Analyzer) merger {
	return merger{finder: finder, analyzer: analyzer}
}

type merger struct {
	finder   Finder
	analyzer Analyzer
}

func (m merger) Merge(name string) error {
	op, err := m.finder.FindPkg(name)
	if err != nil {
		return err
	}

	return m.analyzer.Analyze(op)
}
