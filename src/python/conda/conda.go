package conda

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/kr/text"
)

type Stager interface {
	BuildDir() string
	CacheDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	WriteProfileD(string, string) error
}

type Installer interface {
	InstallOnlyVersion(string, string) error
}

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
}

type Conda struct {
	Installer Installer
	Stager    Stager
	Command   Command
	Log       *libbuildpack.Logger
}

func New(i Installer, s Stager, c Command, l *libbuildpack.Logger) *Conda {
	return &Conda{
		Installer: i,
		Stager:    s,
		Command:   c,
		Log:       l,
	}
}

func Run(c *Conda) error {
	c.Warning()

	if err := c.Install(c.Version()); err != nil {
		c.Log.Error("Could not install conda: %v", err)
		return err
	}

	if err := c.UpdateAndClean(); err != nil {
		c.Log.Error("Could not update conda env: %v", err)
		return err
	}

	c.Stager.LinkDirectoryInDepDir(c.condaBin(), "bin")
	if err := c.Stager.WriteProfileD("conda.sh", c.ProfileD()); err != nil {
		c.Log.Error("Could not write profile.d script: %v", err)
		return err
	}

	c.Log.BeginStep("Done")
	return nil
}

func (c *Conda) Version() string {
	return "miniconda3"
}

func (c *Conda) Install(version string) error {
	c.Log.BeginStep("Supplying conda")
	var installer string
	if installerDir, err := ioutil.TempDir("", "miniconda"); err != nil {
		return err
	} else {
		installer = filepath.Join(installerDir, "miniconda.sh")
		defer os.RemoveAll(installerDir)
	}

	if err := c.Installer.InstallOnlyVersion(version, installer); err != nil {
		return fmt.Errorf("Error downloading miniconda: %v", err)
	}
	if err := os.Chmod(installer, 0755); err != nil {
		return err
	}

	c.Log.BeginStep("Installing Miniconda")
	if err := c.Command.Execute("/", indentWriter(os.Stdout), ioutil.Discard, installer, "-b", "-p", c.condaHome()); err != nil {
		return fmt.Errorf("Error installing miniconda: %v", err)
	}

	return nil
}

func (c *Conda) UpdateAndClean() error {
	c.Log.BeginStep("Installing Dependencies")
	c.Log.BeginStep("Installing conda environment from environment.yml")

	verbosity := []string{"--quiet"}
	if os.Getenv("BP_DEBUG") != "" {
		verbosity = []string{"--debug", "--verbose"}
	}

	condaCache := filepath.Join(c.Stager.CacheDir(), "conda")
	c.Log.Debug("Setting CONDA_PKGS_DIRS to %s", condaCache)
	if err := os.Setenv("CONDA_PKGS_DIRS", condaCache); err != nil {
		return fmt.Errorf("setting CONDA_PKGS_DIRS: %w", err)
	}

	args := append(append([]string{"env", "update"}, verbosity...), "-n", "dep_env", "-f", filepath.Join(c.Stager.BuildDir(), "environment.yml"))
	c.Log.Debug("Run Conda: %s %s", c.condaExec(), strings.Join(args, " "))
	if err := c.Command.Execute("/", indentWriter(os.Stdout), indentWriter(os.Stderr), c.condaExec(), args...); err != nil {
		return fmt.Errorf("Could not run conda env update: %v", err)
	}
	if err := c.Command.Execute("/", indentWriter(os.Stdout), indentWriter(os.Stderr), c.condaExec(), "clean", "-pt"); err != nil {
		c.Log.Error("Could not run conda clean: %v", err)
		return fmt.Errorf("Could not run conda clean: %v", err)
	}

	return nil
}

func (c *Conda) condaHome() string {
	return filepath.Join(c.Stager.DepDir(), "conda")
}

func (c *Conda) condaBin() string {
	return filepath.Join(c.condaHome(), "bin")
}

func (c *Conda) condaExec() string {
	return filepath.Join(c.condaBin(), "conda")
}

func (c *Conda) ProfileD() string {
	return fmt.Sprintf(`grep -rlI %s $DEPS_DIR/%s/conda | xargs sed -i -e "s|%s|$DEPS_DIR/%s|g"
source activate dep_env
`, c.Stager.DepDir(), c.Stager.DepsIdx(), c.Stager.DepDir(), c.Stager.DepsIdx())
}

func (c *Conda) Warning() error {
	if exists, err := libbuildpack.FileExists(filepath.Join(c.Stager.BuildDir(), "runtime.txt")); err != nil {
		return err
	} else if !exists {
		return nil
	}
	if contents, err := ioutil.ReadFile(filepath.Join(c.Stager.BuildDir(), "environment.yml")); err != nil {
		return err
	} else {
		if bytes.Contains(contents, []byte("python=")) {
			c.Log.Warning("you have specified the version of Python runtime both in 'runtime.txt' and 'environment.yml'. You should remove one of the two versions")
		}
	}
	return nil
}

func indentWriter(writer io.Writer) io.Writer {
	return text.NewIndentWriter(writer, []byte("       "))
}
