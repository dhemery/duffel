package file

import (
	"io/fs"
)

// A Root implements [ActionFS] and can supply an [fs.FS] that implements [fs.ReadLinkFS].
type Root interface {
	FS() fs.FS
	ActionFS
}

// A RootFS implements [fs.ReadLinkFS] and [ActionFS] by delegating to a [Root].
type RootFS struct {
	fs.ReadLinkFS
	ActionFS
}

// NewRootFS returns a [RootFS] that delegates to r.
func NewRootFS(r Root) RootFS {
	return RootFS{
		ActionFS:   r,
		ReadLinkFS: r.FS().(fs.ReadLinkFS),
	}
}
