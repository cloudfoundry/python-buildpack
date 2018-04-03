package hooks

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/cloudfoundry/libbuildpack"
)

type AppHook struct {
	libbuildpack.DefaultHook
}

func init() {
	libbuildpack.AddHook(AppHook{})
}

func (h AppHook) BeforeCompile(compiler *libbuildpack.Stager) error {
	return runHook("pre_compile", compiler)
}

func (h AppHook) AfterCompile(compiler *libbuildpack.Stager) error {
	return runHook("post_compile", compiler)
}

func runHook(scriptName string, compiler *libbuildpack.Stager) error {
	path := filepath.Join(compiler.BuildDir(), "bin", scriptName)
	if exists, err := libbuildpack.FileExists(path); err != nil {
		return err
	} else if exists {
		compiler.Logger().BeginStep("Running " + scriptName + " hook")
		if err := os.Chmod(path, 0755); err != nil {
			return err
		}

		fileContents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		shebangRegex := regexp.MustCompile("^\\s*#!")
		hasShebang := shebangRegex.Match(fileContents)

		var cmd *exec.Cmd
		if hasShebang {
			cmd = exec.Command(path)
		} else {
			cmd = exec.Command("/bin/sh", path)
		}

		cmd.Dir = compiler.BuildDir()
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		compiler.Logger().Info("%s", output)
	}
	return nil

}
