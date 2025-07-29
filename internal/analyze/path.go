package analyze

import (
	"path"
	"path/filepath"
)

// NewSourcePath returns a new SourcePath that represents
// a package or item in a duffel source tree.
// Source is the full path to the source directory.
// Pkg is the name of the package directory.
// Item is the path from the package directory to the item.
// If item is empty, the SourcePath represents a package.
func NewSourcePath(source, pkg, item string) SourcePath {
	return SourcePath{source, pkg, item}
}

// SourcePath reresents the path to a package or item in a duffel source tree.
type SourcePath struct {
	s string // The full path to the source directory.
	p string // The name of the package.
	i string // The path from the package directory to the item.
}

// String returns the full path to s.
func (s SourcePath) String() string {
	return path.Join(s.s, s.p, s.i)
}

// PackageDir returns the full path to s;s package directory.
func (s SourcePath) PackageDir() string {
	return path.Join(s.s, s.p)
}

func (s SourcePath) Item() string {
	return s.i
}

// WithItem returns a copy of s with its item replaced by item.
func (s SourcePath) WithItem(item string) SourcePath {
	return SourcePath{s.s, s.p, item}
}

// WithItemFrom returns a copy of s with its item replaced by the item in name.
// Name must be in the same package as s.
func (s SourcePath) WithItemFrom(name string) SourcePath {
	item, _ := filepath.Rel(s.PackageDir(), name)
	return s.WithItem(item)
}

func NewTargetPath(target, item string) TargetPath {
	return TargetPath{target, item}
}

// TargetPath represents the path to a file in the target tree
// that corresponds to a item in the source tree.
type TargetPath struct {
	t string // The full path to the target directory.
	i string // The path to the item from target.
}

// String returns the full path to the item.
func (t TargetPath) String() string {
	return path.Join(t.t, t.i)
}

// Rel returns the relative path to follow to reach full from t.
func (t TargetPath) Rel(full string) (string, error) {
	return filepath.Rel(path.Dir(t.String()), full)
}

// Full returns the full path reached by following rel from t.
func (t TargetPath) Full(rel string) string {
	return path.Join(t.parent(), rel)
}

func (t TargetPath) parent() string {
	return path.Join(t.t, path.Dir(t.i))
}
