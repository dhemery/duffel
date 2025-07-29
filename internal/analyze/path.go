package analyze

import (
	"path"
	"path/filepath"
)

// NewSourceItem returns a new SourceItem that represents
// a package or item in a duffel source tree.
// Source is the full path to the source directory.
// Pkg is the name of the package directory.
// Item is the path from the package directory to the item.
func NewSourceItem(source, pkg, item string) SourceItem {
	return SourceItem{source, pkg, item}
}

// SourceItem reresents the path to a package or item in a duffel source tree.
type SourceItem struct {
	Source  string // The full path to the source directory.
	Package string // The name of the package.
	Item    string // The path from the package directory to the item.
}

// String returns the full path to the package or item represented by s.
func (s SourceItem) String() string {
	return path.Join(s.Source, s.Package, s.Item)
}

// PackageDir returns the full path to the package directory represented by s.
func (s SourceItem) PackageDir() string {
	return path.Join(s.Source, s.Package)
}

// WithItem returns a copy of s with its item replaced by item.
func (s SourceItem) WithItem(item string) SourceItem {
	wi := s
	wi.Item = item
	return wi
}

// WithItemFrom returns a copy of s with its item replaced by the item in name.
// Name must be in the same package as s.
func (s SourceItem) WithItemFrom(name string) SourceItem {
	item, _ := filepath.Rel(s.PackageDir(), name)
	return SourceItem{
		Source:  s.Source,
		Package: s.Package,
		Item:    item,
	}
}

// TargetItem represents the path to a file in the target tree
// that corresponds to a item in the source tree.
type TargetItem struct {
	Target string // The full path to the root of the target tree.
	Item   string // The path to the item from target.
}

// String returns the full path to the target item.
func (t TargetItem) String() string {
	return path.Join(t.Target, t.Item)
}

// Rel returns the relative path to follow to reach full from t.
func (t TargetItem) Rel(full string) (string, error) {
	return filepath.Rel(path.Dir(t.String()), full)
}

// Full returns the full path reached by following rel from t.
func (t TargetItem) Full(rel string) string {
	return path.Join(t.parent(), rel)
}

func (t TargetItem) parent() string {
	return path.Join(t.Target, path.Dir(t.Item))
}
