package plan

type Finder interface {
	FindPkg(name string) (PkgOp, error)
}

type Planner interface {
	Plan(ops ...PkgOp) error
}

func NewMerger(finder Finder, planner Planner) merger {
	return merger{finder: finder, planner: planner}
}

type merger struct {
	finder  Finder
	planner Planner
}

func (m merger) Merge(name string) error {
	op, err := m.finder.FindPkg(name)
	if err != nil {
		return err
	}

	return m.planner.Plan(op)
}
