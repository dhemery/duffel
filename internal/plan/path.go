package plan

import (
	"encoding/json/jsontext"
	"path"
	"path/filepath"

	"github.com/dhemery/duffel/internal/file"
)

// newSourcePath returns a [sourcePath]
// for the specified package or item.
// Source is the full path from the root of the file system to the source directory.
// Pkg is the name of the package directory.
// Item is the path from the package directory to the item.
// If item is empty, the SourcePath represents a package.
func newSourcePath(source, pkg, item string) sourcePath {
	return sourcePath{source, pkg, item}
}

// A sourcePath is the path to a package or item in a duffel source tree.
type sourcePath struct {
	source string // The full path to the source directory.
	pkg    string // The name of the package.
	item   string // The path from the package directory to the item.
}

// String returns the full path to s.
func (s sourcePath) String() string {
	return path.Join(s.source, s.pkg, s.item)
}

// MarshalJSONTo writes the string value of s to e.
func (s sourcePath) MarshalJSONTo(e *jsontext.Encoder) error {
	return e.WriteToken(jsontext.String(s.String()))
}

// PackageDir returns the full path to s's package directory.
func (s sourcePath) PackageDir() string {
	return path.Join(s.source, s.pkg)
}

// withItem returns a copy of s with its item replaced by item.
func (s sourcePath) withItem(item string) sourcePath {
	return sourcePath{s.source, s.pkg, item}
}

// withItemFrom returns a copy of s with its item replaced by the item in name.
// Name must be in the same package as s.
func (s sourcePath) withItemFrom(name string) sourcePath {
	item, _ := filepath.Rel(s.PackageDir(), name)
	return s.withItem(item)
}

// newSourceItem returns a [sourceItem] with the given path and file type.
func newSourceItem(source, pkg, item string, t file.Type) sourceItem {
	return sourceItem{newSourcePath(source, pkg, item), t}
}

// A sourceItem describes a file in a duffel source tree.
type sourceItem struct {
	Path sourcePath `json:"path"` // The path to the file.
	Type file.Type  `json:"type"` // The type of the file.
}

// newTargetPath returns a [targetPath]
// for the specified item in the target tree.
func newTargetPath(target, item string) targetPath {
	return targetPath{target, item}
}

// A targetPath is the path to an existing or planned file in the target tree.
type targetPath struct {
	target string // The full path to the target directory.
	item   string // The path from t to the file.
}

// String returns the full path to the file.
func (t targetPath) String() string {
	return path.Join(t.target, t.item)
}

// MarshalJSONTo writes the string value of t to e.
func (t targetPath) MarshalJSONTo(e *jsontext.Encoder) error {
	return e.WriteToken(jsontext.String(t.String()))
}

// PathTo returns the relative path to full from t's parent directory.
func (t targetPath) PathTo(full string) string {
	p, _ := filepath.Rel(t.parent(), full)
	return p
}

// Resolve resolves rel with respect to t's parent directory.
func (t targetPath) Resolve(rel string) string {
	return path.Join(t.parent(), rel)
}

func (t targetPath) parent() string {
	return path.Dir(t.String())
}

// newTargetItem returns a [targetItem] with the given path and file state.
func newTargetItem(target, item string, state file.State) targetItem {
	return targetItem{targetPath{target, item}, state}
}

// A targetItem describes a planned or existing file in the target tree.
type targetItem struct {
	Path  targetPath `json:"path"`  // The path to the file.
	State file.State `json:"state"` // The state of the file.
}
