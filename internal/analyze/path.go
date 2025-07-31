package analyze

import (
	"io/fs"
	"log/slog"
	"path"
	"path/filepath"

	"github.com/dhemery/duffel/internal/file"
)

// NewSourcePath returns a [SourcePath]
// for the specified package or item.
// Source is the full path to the source directory.
// Pkg is the name of the package directory.
// Item is the path from the package directory to the item.
// If item is empty, the SourcePath represents a package.
func NewSourcePath(source, pkg, item string) SourcePath {
	return SourcePath{source, pkg, item}
}

// SourcePath is the path to a package or item in duffel source tree.
type SourcePath struct {
	Source  string `json:"source"`  // The full path to the source directory.
	Package string `json:"package"` // The name of the package.
	Item    string `json:"item"`    // The path from the package directory to the item.
}

// String returns the full path to s.
func (s SourcePath) String() string {
	return path.Join(s.Source, s.Package, s.Item)
}

// PackageDir returns the full path to s;s package directory.
func (s SourcePath) PackageDir() string {
	return path.Join(s.Source, s.Package)
}

// WithItem returns a copy of s with its item replaced by item.
func (s SourcePath) WithItem(item string) SourcePath {
	return SourcePath{s.Source, s.Package, item}
}

// WithItemFrom returns a copy of s with its item replaced by the item in name.
// Name must be in the same package as s.
func (s SourcePath) WithItemFrom(name string) SourcePath {
	item, _ := filepath.Rel(s.PackageDir(), name)
	return s.WithItem(item)
}

type SourceItem struct {
	Path  SourcePath  `json:"path"`
	Entry fs.DirEntry `json:"entry"`
}

func (s SourceItem) Equal(o SourceItem) bool {
	return s.Path == o.Path &&
		s.Entry.Type() == o.Entry.Type()
}

func (s SourceItem) LogValue() slog.Value {
	return slog.GroupValue(slog.Any("path", s.Path),
		slog.String("entry", fs.FormatDirEntry(s.Entry)))
}

// NewTargetPath returns a [TargetPath]
// for the specified item in the target tree.
func NewTargetPath(target, item string) TargetPath {
	return TargetPath{target, item}
}

// TargetPath is the path to an existing or planned file in the target tree.
type TargetPath struct {
	Target string `json:"target"` // The full path to the target directory.
	Item   string `json:"item"`   // The path from t to the file.
}

// String returns the full path to the item.
func (t TargetPath) String() string {
	return path.Join(t.Target, t.Item)
}

// PathTo returns the relative path to full from t's parent directory.
func (t TargetPath) PathTo(full string) string {
	p, _ := filepath.Rel(t.parent(), full)
	return p
}

// Resolve resolves rel with respect to t's parent directory.
func (t TargetPath) Resolve(rel string) string {
	return path.Join(t.parent(), rel)
}

func (t TargetPath) parent() string {
	return path.Dir(t.String())
}

type TargetItem struct {
	Path  TargetPath  `json:"path"`
	State *file.State `json:"state"`
}

func (t TargetItem) Equal(o TargetItem) bool {
	return t.Path == o.Path &&
		t.State.Equal(o.State)
}

func (t TargetItem) LogValue() slog.Value {
	return slog.GroupValue(slog.Any("path", t.Path),
		slog.Any("state", t.State))
}
