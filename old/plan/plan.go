package plan

import (
	"io/fs"
)

type Planner struct {
	FS          fs.FS
	FarmRoot    string
	InstallRoot string
	Packages    []string
}

type Plan struct {
	Links    []string
	Problems []error
}
