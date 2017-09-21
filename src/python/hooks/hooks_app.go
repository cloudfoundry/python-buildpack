package hooks

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type AppHook struct {
	libbuildpack.DefaultHook
}

func init() {
	libbuildpack.AddHook(AppHook{})
}

func (h AppHook) BeforeCompile(compiler *libbuildpack.Stager) error {
	path := filepath.Join(compiler.BuildDir(), "bin", "pre_compile")
	if exists, err := libbuildpack.FileExists(path); err != nil {
		return err
	} else if exists {
		compiler.Logger().BeginStep("Running pre-compile hook")
		if err := os.Chmod(path, 0755); err != nil {
			return err
		}
		cmd := exec.Command("/bin/sh", path)
		cmd.Dir = compiler.BuildDir()
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		compiler.Logger().Info("%s", output)

	}
	return nil
}

func (h AppHook) AfterCompile(compiler *libbuildpack.Stager) error {
	path := filepath.Join(compiler.BuildDir(), "bin", "post_compile")
	if exists, err := libbuildpack.FileExists(path); err != nil {
		return err
	} else if exists {
		compiler.Logger().BeginStep("Running post-compile hook")
		if err := os.Chmod(path, 0755); err != nil {
			return err
		}
		cmd := exec.Command("/bin/sh", path)
		cmd.Dir = compiler.BuildDir()
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		compiler.Logger().Info("%s", output)
	}
	return nil
}
