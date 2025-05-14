package plan

import "fmt"

type InstallAdvisor struct {
	Target FS
	Source FS
	Prefix string
}

func NewInstallAdvisor(source FS, target FS, prefix string) *InstallAdvisor {
	return &InstallAdvisor{
		Target: target,
		Source: source,
		Prefix: prefix,
	}
}

func (a *InstallAdvisor) Advise(plan *Plan, item string) (*Action, error) {
	return nil, nil
}

func (a *InstallAdvisor) String() string {
	return fmt.Sprintf("install packages from %s to %s with prefix %s", a.Source, a.Target, a.Prefix)
}
