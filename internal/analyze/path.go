package analyze

import (
	"io/fs"
	"path"
	"path/filepath"
)

// SourceItem is the path to an item in a duffel source tree.
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
// Name must be in the same package as s.
func (s SourceItem) WithItemFrom(name string) (SourceItem, error) {
	item, err := filepath.Rel( s.PackageDir(), name)
	if err != nil {
		return SourceItem{}, err
	}
	if !fs.ValidPath(item) {
		return SourceItem{}, fs.ErrInvalid
	}

	return s.WithItem(item), nil
}
