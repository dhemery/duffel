package plan

import (
	"fmt"
	"io/fs"
)

type FS interface {
	fs.ReadDirFS
	Lstat(name string) (fs.FileInfo, error)
	ReadLink(name string) (string, error)
}

type Advisor interface {
	Advise(plan *Plan, item string) (*Action, error)
}

type Action interface {
	Error() error
}

type Plan struct {
	Actions map[string]*Action
	Errors  []error
}

func Build(source FS, advisor Advisor, packages []string) *Plan {
	fmt.Println("Building plan to", advisor)
	fmt.Println("Packages", packages, "from source", source)
	return nil
}
