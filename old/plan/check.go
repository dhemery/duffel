package plan

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"dhemery.com/duffel/rules"
)

func MakePlan(duffelPath string, installPath string, packages []string) error {
	fsys, err := findFS(duffelPath, installPath)
	if err != nil {
		return err
	}

	if err := rules.CheckIsDuffelDir(fsys, duffelPath); err != nil {
		return fmt.Errorf("invalid duffel path %s: %w", installPath, err)
	}
	if err := rules.CheckInstallPath(fsys, installPath); err != nil {
		return fmt.Errorf("invalid install path %s: %w", installPath, err)
	}

	for _, name := range packages {
		packagePath := path.Join(duffelPath, name)
		if err := rules.CheckPackagePath(fsys, packagePath); err != nil {
			return fmt.Errorf("invalid package path %s: %w", packagePath, err)
		}
	}
	return nil
}

func findFS(a, b string) (fs.FS, error) {
	return nil, nil
}

func checkFiles(fsys fs.FS) error {
	_ = fs.WalkDir(fsys, ".", checkFile)
	return errors.New("just checking")
}

func checkFile(path string, entry fs.DirEntry, errIn error) error {
	fmt.Println(path, entry)
	fmt.Println("   is dir", entry.IsDir())
	mode := entry.Type()
	fmt.Println("   mode", mode)
	fmt.Println("   mode is dir", mode.IsDir())
	fmt.Println("   mode is regular", mode.IsRegular())
	fmt.Println("   mode dir bit", mode&fs.ModeDir != 0)
	fmt.Println("   mode link bit", mode&fs.ModeSymlink != 0)
	fmt.Println()

	return nil
}
