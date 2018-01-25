package conda

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
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

type Manifest interface {
	InstallOnlyVersion(string, string) error
}

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
}

type Conda struct {
	Manifest Manifest
	Stager   Stager
	Command  Command
	Log      *libbuildpack.Logger
}

func New(m Manifest, s Stager, c Command, l *libbuildpack.Logger) *Conda {
	return &Conda{
		Manifest: m,
		Stager:   s,
		Command:  c,
		Log:      l,
	}
}

func Run(c *Conda) error {
	c.Warning()

	if err := c.Install(c.Version()); err != nil {
		c.Log.Error("Could not install conda: %v", err)
		return err
	}

	if err := c.RestoreCache(); err != nil {
		c.Log.Error("Could not restore conda envs cache: %v", err)
		return err
	}

	if err := c.UpdateAndClean(); err != nil {
		c.Log.Error("Could not update conda env: %v", err)
		return err
	}

	if err := c.SaveCache(); err != nil {
		c.Log.Error("Could not save conda envs cache: %v", err)
		return err
	}

	c.Stager.LinkDirectoryInDepDir(filepath.Join(c.Stager.DepDir(), "conda", "bin"), "bin")
	if err := c.Stager.WriteProfileD("conda.sh", c.ProfileD()); err != nil {
		c.Log.Error("Could not write profile.d script: %v", err)
		return err
	}

	c.Log.BeginStep("Done")
	return nil
}

func (c *Conda) Version() string {
	if runtime, err := ioutil.ReadFile(filepath.Join(c.Stager.BuildDir(), "runtime.txt")); err == nil {
		if strings.HasPrefix(string(runtime), "python-3") {
			return "miniconda3"
		}
	}
	return "miniconda2"
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

	if err := c.Manifest.InstallOnlyVersion(version, installer); err != nil {
		return fmt.Errorf("Error downloading miniconda: %v", err)
	}
	if err := os.Chmod(installer, 0755); err != nil {
		return err
	}

	c.Log.BeginStep("Installing Miniconda")
	condaHome := filepath.Join(c.Stager.DepDir(), "conda")
	if err := c.Command.Execute("/", indentWriter(os.Stdout), ioutil.Discard, installer, "-b", "-p", condaHome); err != nil {
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

	condaHome := filepath.Join(c.Stager.DepDir(), "conda")
	args := append(append([]string{"env", "update"}, verbosity...), "-n", "dep_env", "-f", filepath.Join(c.Stager.BuildDir(), "environment.yml"))
	c.Log.Debug("Run Conda: %s %s", filepath.Join(condaHome, "bin", "conda"), strings.Join(args, " "))
	if err := c.Command.Execute("/", indentWriter(os.Stdout), indentWriter(os.Stderr), filepath.Join(condaHome, "bin", "conda"), args...); err != nil {
		return fmt.Errorf("Could not run conda env update: %v", err)
	}
	if err := c.Command.Execute("/", indentWriter(os.Stdout), indentWriter(os.Stderr), filepath.Join(condaHome, "bin", "conda"), "clean", "-pt"); err != nil {
		c.Log.Error("Could not run conda clean: %v", err)
		return fmt.Errorf("Could not run conda clean: %v", err)
	}

	return nil
}

func (c *Conda) ProfileD() string {
	return fmt.Sprintf(`grep -rlI %s $DEPS_DIR/%s/conda | xargs sed -i -e "s|%s|$DEPS_DIR/%s|g"
source activate dep_env
`, c.Stager.DepDir(), c.Stager.DepsIdx(), c.Stager.DepDir(), c.Stager.DepsIdx())
}

func (c *Conda) SaveCache() error {
	if err := os.MkdirAll(filepath.Join(c.Stager.CacheDir(), "envs"), 0755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(c.Stager.CacheDir(), "conda_prefix"), []byte(c.Stager.DepDir()), 0644); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(c.Stager.CacheDir(), "envs")); err != nil {
		return err
	}

	if output, err := c.Command.Output("/", "cp", "-Rl", filepath.Join(c.Stager.DepDir(), "conda", "envs"), filepath.Join(c.Stager.CacheDir(), "envs")); err != nil {
		return fmt.Errorf("%s\n%v", output, err)
	}

	return nil
}

func (c *Conda) RestoreCache() error {
	if err := os.MkdirAll(filepath.Join(c.Stager.DepDir(), "conda", "envs"), 0755); err != nil {
		return err
	}
	dirs, err := filepath.Glob(filepath.Join(c.Stager.CacheDir(), "envs", "*"))
	if err != nil {
		return err
	}

	if len(dirs) > 0 {
		c.Log.BeginStep("Using dependency cache at %s", filepath.Join(c.Stager.CacheDir(), "envs"))
	}

	for _, dir := range dirs {
		os.Rename(dir, filepath.Join(c.Stager.DepDir(), "conda", "envs", path.Base(dir)))
	}

	return c.restoreCacheRewriteOldDepDir()
}

func (c *Conda) restoreCacheRewriteOldDepDir() error {
	bPrefix, err := ioutil.ReadFile(filepath.Join(c.Stager.CacheDir(), "conda_prefix"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	prefix := strings.TrimSpace(string(bPrefix))

	// grep -rlI $prefix $DEPS_DIR/$DEPS_IDX/conda | xargs sed -i -e "s|$prefix|$DEPS_DIR/$DEPS_IDX|g"
	if err := filepath.Walk(filepath.Join(c.Stager.DepDir(), "conda", "envs"), func(path string, info os.FileInfo, err error) error {
		if !info.Mode().IsRegular() {
			return nil
		}
		if contents, err := ioutil.ReadFile(path); err != nil {
			return fmt.Errorf("Readfile: %v", err)
		} else {
			if bytes.Contains(contents, []byte(prefix)) {
				contents = bytes.Replace(contents, []byte(prefix), []byte(c.Stager.DepDir()), -1)
				if err := ioutil.WriteFile(path, contents, 0644); err != nil {
					return fmt.Errorf("WriteFile: %v", err)
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
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
