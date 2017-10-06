package pyfinder

import (
	"fmt"
	"path/filepath"
)

type ManagePyFinder struct{}

func (m ManagePyFinder) FindManagePy(dir string) (string, error) {
	for _, glob := range []string{"manage.py", "*/manage.py", "*/*/manage.py"} {
		if matches, err := filepath.Glob(filepath.Join(dir, glob)); err != nil {
			return "", fmt.Errorf("Finding %s: %v", glob, err)
		} else if len(matches) > 0 {
			return matches[0], nil
		}
	}

	return "", fmt.Errorf("manage.py not found!")
}
