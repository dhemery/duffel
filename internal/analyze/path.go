package analyze

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
)

// SourceItem is the path to a nckage or item in a duffel source tree.
// If Item is "", the SourceItem represents a package.
type SourceItem struct {
	Source  string // The full path to the source directory that contains the item.
	Package string // The name of the package directory that contains the item.
	Item    string // The item's path relative to the package directory.
}

// String returns the full path to the item.
func (s SourceItem) String() string {
	return path.Join(s.Source, s.Package, s.Item)
}

// PackageDir returns the path to the package directory that contains the item.
func (s SourceItem) PackageDir() string {
	return path.Join(s.Source, s.Package)
}

// WithItem returns a copy of s with its Item replaced by item.
func (s SourceItem) WithItem(item string) SourceItem {
	wi := s
	wi.Item = item
	return wi
}

// WithItemFrom returns a copy of s with its Item replaced by the item in name.
// If name is in not the same package as s, the method panics.
func (s SourceItem) WithItemFrom(name string) SourceItem {
	item, err := filepath.Rel(s.PackageDir(), name)
	if err != nil || !fs.ValidPath(item) {
		panic(fmt.Errorf("PackageItem.WithItemFrom(%q) called with arg not in same package as receiver %q",
			name, s))
	}
	return s.WithItem(item)
}
