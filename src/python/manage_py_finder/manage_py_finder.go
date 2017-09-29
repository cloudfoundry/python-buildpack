package manage_py_finder

import (
	"fmt"
	"path/filepath"
)

type ManagePyFinder struct{}

func (m ManagePyFinder) FindManagePy(dir string) (string, error) {

	matches, _ := filepath.Glob(filepath.Join(dir, "manage.py"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	matches, _ = filepath.Glob(filepath.Join(dir, "*/manage.py"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	matches, _ = filepath.Glob(filepath.Join(dir, "*/*/manage.py"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	return "", fmt.Errorf("manage.py not found!")
}
