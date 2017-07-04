package supply

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
}

type Manifest interface {
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type Stager interface {
	BuildDir() string
	CacheDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	WriteEnvFile(string, string) error
	WriteProfileD(string, string) error
}

type Supplier struct {
	Stager   Stager
	Manifest Manifest
	Log      *libbuildpack.Logger
	Command  Command
}

func Run(s *Supplier) error {
	if os.Getenv("LANG") == "" {
		os.Setenv("LANG", "en_US.UTF-8")
	}

	// TODO: Conda

	if err := s.WarnNoStart(); err != nil {
		s.Log.Error("Unable to test procfile existence: %s", err.Error())
		return err
	}

	if err := s.RestoreCache(); err != nil {
		s.Log.Error("Unable to restore cache: %s", err.Error())
		return err
	}

	// TODO App based hooks

	// TODO https://github.com/cloudfoundry/python-buildpack/blob/master/bin/steps/pipenv-python-version.rb

	if err := s.InstallPython(); err != nil {
		s.Log.Error("Unable to install python: %s", err.Error())
		return err
	}
	if err := s.InstallPip(); err != nil {
		s.Log.Error("Unable to install pip: %s", err.Error())
		return err
	}

	for _, name := range []string{"pip-pop", "pipenv"} {
		if err := s.InstallViaPip(name); err != nil {
			s.Log.Error("Unable to install %s: %s", name, err.Error())
			return err
		}
	}

	// TODO "Generating 'requirements.txt' with pipenv"
	// https://github.com/cloudfoundry/python-buildpack/blob/1588bd4099b9b2f75448c62bcea80e38dd5795a8/bin/steps/pipenv#L16-L30

	if err := s.InstallCryptography(); err != nil {
		s.Log.Error("Unable to install pip: %s", err.Error())
		return err
	}

	if err := s.InstallPylibmc(); err != nil {
		s.Log.Error("Unable to install pip: %s", err.Error())
		return err
	}

	if err := s.CreateDefaultEnv(); err != nil {
		s.Log.Error("Unable to setup default environment: %s", err.Error())
		return err
	}

	return nil
}

func (s *Supplier) WarnNoStart() error {
	procfileExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "Procfile"))
	if err != nil {
		return err
	}

	if !procfileExists {
		warning := "Your application is missing a Procfile. This file tells Cloud Foundry how to run your application.\n"
		warning += "Learn more: https://docs.cloudfoundry.org/buildpacks/prod-server.html#procfile"
		s.Log.Warning(warning)
	}

	return nil
}

// Move cache dirs to build dir (will copy back after build)
func (s *Supplier) RestoreCache() error {
	for _, name := range []string{"python", "src"} {
		srcDirExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.CacheDir(), name))
		if err != nil {
			return err
		}
		if srcDirExists {
			if err := os.Rename(filepath.Join(s.Stager.CacheDir(), name), filepath.Join(s.Stager.DepDir(), name)); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Supplier) InstallPython() error {
	var err error
	dep := libbuildpack.Dependency{Name: "python"}

	installDir := filepath.Join(s.Stager.DepDir(), "python")

	// TODO set from runtime.txt if exists
	desiredVersion := ""

	if desiredVersion != "" {
		versions := s.Manifest.AllDependencyVersions(dep.Name)
		dep.Version, err = libbuildpack.FindMatchingVersion(desiredVersion, versions)
		if err != nil {
			return err
		}
	} else {
		dep, err = s.Manifest.DefaultVersion("python")
		if err != nil {
			return err
		}
	}

	if err := s.Manifest.InstallDependency(dep, installDir); err != nil {
		return err
	}

	if err := s.symlinkAll([]string{"python"}); err != nil {
		return err
	}

	// TODO Record for future reference.
	// echo $PYTHON_VERSION > "$CACHE_DIR/python-version"
	// echo $CF_STACK > "$CACHE_DIR/python-stack"

	if err := os.Setenv("PYTHONPATH", s.Stager.DepDir()); err != nil {
		return err
	}
	return os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), filepath.Join(s.Stager.DepDir(), "bin")))
}

func (s *Supplier) InstallPip() error {
	// TODO Only do if required -- https://github.com/cloudfoundry/python-buildpack/blob/master/bin/steps/python

	dir, err := ioutil.TempDir("", "setuptools")
	if err != nil {
		return err
	}

	if err := s.Manifest.InstallOnlyVersion("setuptools", dir); err != nil {
		return err
	}

	matches, err := filepath.Glob(filepath.Join(dir, "setuptools-*"))
	if err != nil || len(matches) != 1 {
		return errors.New("Could not find expected extracted directory")
	}

	if err := s.Command.Execute(matches[0], ioutil.Discard, ioutil.Discard, "python", "setup.py", "install", "--prefix="+filepath.Join(s.Stager.DepDir(), "python")); err != nil {
		return err
	}

	// TODO refactor this since above and below are the same

	dir, err = ioutil.TempDir("", "pip")
	if err != nil {
		return err
	}

	if err := s.Manifest.InstallOnlyVersion("pip", dir); err != nil {
		return err
	}

	matches, err = filepath.Glob(filepath.Join(dir, "pip-*"))
	if err != nil || len(matches) != 1 {
		return errors.New("Could not find expected extracted directory")
	}

	if err := s.Command.Execute(matches[0], ioutil.Discard, ioutil.Discard, "python", "setup.py", "install", "--prefix="+filepath.Join(s.Stager.DepDir(), "python")); err != nil {
		return err
	}

	return s.symlinkAll([]string{"python"})
}

func (s *Supplier) symlinkAll(names []string) error {
	for _, name := range names {
		installDir := filepath.Join(s.Stager.DepDir(), name)

		for _, dir := range []string{"bin", "lib", "include", "pkgconfig", "lib/pkgconfig"} {
			exists, err := libbuildpack.FileExists(filepath.Join(installDir, dir))
			if err != nil {
				return err
			}
			if exists {
				if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(installDir, dir), path.Base(dir)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Supplier) InstallViaPip(name string) error {
	dir, err := ioutil.TempDir("", name)
	if err != nil {
		return err
	}

	if err := s.Manifest.InstallOnlyVersion(name, dir); err != nil {
		return err
	}

	if err := s.Command.Execute("", ioutil.Discard, ioutil.Discard, "pip", "install", name, "--exists-action=w", "--no-index", "--find-links="+dir); err != nil {
		return err
	}

	return s.symlinkAll([]string{"python"})
}

func (s *Supplier) pipGrepHas(name string) (bool, error) {
	if err := s.Command.Execute(s.Stager.BuildDir(), ioutil.Discard, ioutil.Discard, "pip-grep", "-s", "requirements.txt", name); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *Supplier) InstallPylibmc() error {
	hasPylibmc, err := s.pipGrepHas("pylibmc")
	if err != nil {
		return err
	}

	if hasPylibmc {
		s.Log.Info("Noticed pylibmc. Bootstrapping libmemcached.")
		installDir := filepath.Join(s.Stager.DepDir(), "libmemcache")

		if err := s.Manifest.InstallOnlyVersion("libmemcache", installDir); err != nil {
			return err
		}

		if err := os.Setenv("LIBMEMCACHED", installDir); err != nil {
			return err
		}

		if err := s.Stager.WriteEnvFile("LIBMEMCACHED", installDir); err != nil {
			return err
		}

		if err := s.symlinkAll([]string{"libmemcache"}); err != nil {
			return err
		}

		return s.Stager.LinkDirectoryInDepDir(filepath.Join(installDir, "lib/sasl2"), "lib")
	}

	return nil
}

func (s *Supplier) InstallCryptography() error {
	// TODO needs to check any of
	// argon2-cffi bcrypt cffi cryptography django[argon2] Django[argon2] django[bcrypt] Django[bcrypt] PyNaCl pyOpenSSL PyOpenSSL requests[security] misaka

	needsCrypto, err := s.pipGrepHas("cffi")
	if err != nil {
		return err
	}

	if needsCrypto {
		s.Log.Info("Noticed pylibmc. Bootstrapping libffi.")
		installDir := filepath.Join(s.Stager.DepDir(), "libffi")

		if err := s.Manifest.InstallOnlyVersion("libffi", installDir); err != nil {
			return err
		}

		if err := os.Setenv("LIBFFI", installDir); err != nil {
			return err
		}

		if err := s.Stager.WriteEnvFile("LIBFFI", installDir); err != nil {
			return err
		}

		return s.symlinkAll([]string{"libffi", "libffi/lib/libffi-3.2.1"})
	}

	return nil
}

func (s *Supplier) CreateDefaultEnv() error {
	if err := s.Stager.WriteProfileD("python.gunicorn.sh", "'*'"); err != nil {
		return err
	}

	var environmentDefaults = map[string]string{
		"PYTHONHASHSEED":   "random",
		"PYTHONPATH":       s.Stager.DepDir(),
		"LANG":             os.Getenv("LANG"),
		"PYTHONHOME":       filepath.Join(s.Stager.DepDir(), "python"),
		"PYTHONUNBUFFERED": "1",
	}
	for envVar, envDefault := range environmentDefaults {
		if os.Getenv(envVar) == "" {
			if err := s.Stager.WriteEnvFile(envVar, envDefault); err != nil {
				return err
			}
		}
	}

	if err := s.Stager.WriteProfileD("python.gunicorn.sh", "export FORWARDED_ALLOW_IPS='*'\n"); err != nil {
		return err
	}

	scriptContents := `
export PYTHONHASHSEED=${PYTHONHASHSEED:-random}
export PYTHONPATH=${PYTHONPATH:-$DEPS_DIR/%s}
export LANG=${LANG:-en_US.UTF-8}
export PYTHONHOME=${PYTHONHOME:-$DEPS_DIR/%s/python}
export PYTHONUNBUFFERED=${PYTHONUNBUFFERED:-1}
`

	return s.Stager.WriteProfileD("python.supply.sh", fmt.Sprintf(scriptContents, s.Stager.DepsIdx(), s.Stager.DepsIdx()))
}
