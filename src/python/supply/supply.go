package supply

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
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
	// DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	// WriteEnvFile(string, string) error
	// WriteProfileD(string, string) error
}

type Supplier struct {
	Stager   Stager
	Manifest Manifest
	Log      *libbuildpack.Logger
	Command  Command
}

func Run(s *Supplier) error {
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

	// s.Log.BeginStep("Installing binaries")
	// if err := s.LoadPackageJSON(); err != nil {
	// 	s.Log.Error("Unable to load package.json: %s", err.Error())
	// 	return err
	// }

	// s.WarnNodeEngine()

	// if err := s.InstallNode("/tmp/node"); err != nil {
	// 	s.Log.Error("Unable to install node: %s", err.Error())
	// 	return err
	// }

	// if err := s.InstallNPM(); err != nil {
	// 	s.Log.Error("Unable to install npm: %s", err.Error())
	// 	return err
	// }

	// if err := s.InstallYarn(); err != nil {
	// 	s.Log.Error("Unable to install yarn: %s", err.Error())
	// 	return err
	// }

	// if err := s.CreateDefaultEnv(); err != nil {
	// 	s.Log.Error("Unable to setup default environment: %s", err.Error())
	// 	return err
	// }

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
	if err := os.Rename(filepath.Join(s.Stager.CacheDir(), "python"), filepath.Join(s.Stager.DepDir(), "python")); err != nil {
		return err
	}

	srcDirExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.CacheDir(), "src"))
	if err != nil {
		return err
	}
	if srcDirExists {
		if err := os.Rename(filepath.Join(s.Stager.CacheDir(), "src"), filepath.Join(s.Stager.DepDir(), "src")); err != nil {
			return err
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

	for _, dir := range []string{"bin", "lib", "include", "pkgconfig"} {
		if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(installDir, dir), dir); err != nil {
			return err
		}
	}

	// TODO Record for future reference.
	// echo $PYTHON_VERSION > "$CACHE_DIR/python-version"
	// echo $CF_STACK > "$CACHE_DIR/python-stack"

	return os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("PATH"), filepath.Join(s.Stager.DepDir(), "bin")))
}

func (s *Supplier) InstallPip() error {
	// TODO Only do if required -- https://github.com/cloudfoundry/python-buildpack/blob/master/bin/steps/python

	s.Log.BeginStep("Installing Setuptools")

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

	if err := s.Command.Execute(matches[0], ioutil.Discard, ioutil.Discard, "python", "setup.py", "install", "--install="+filepath.Join(s.Stager.DepDir(), "python")); err != nil {
		return err
	}

	// TODO refactor this since above and below are the same

	s.Log.BeginStep("Installing Pip")

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

	if err := s.Command.Execute(matches[0], ioutil.Discard, ioutil.Discard, "python", "setup.py", "install", "--install="+filepath.Join(s.Stager.DepDir(), "python")); err != nil {
		return err
	}

	installDir := filepath.Join(s.Stager.DepDir(), "python")

	for _, dir := range []string{"bin", "lib", "include", "pkgconfig"} {
		if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(installDir, dir), dir); err != nil {
			return err
		}
	}

	return nil
}

// func (s *Supplier) CreateDefaultEnv() error {
// 	var environmentDefaults = map[string]string{
// 		"NODE_ENV":              "production",
// 		"NPM_CONFIG_PRODUCTION": "true",
// 		"NPM_CONFIG_LOGLEVEL":   "error",
// 		"NODE_MODULES_CACHE":    "true",
// 		"NODE_VERBOSE":          "false",
// 	}

// 	s.Log.BeginStep("Creating runtime environment")

// 	for envVar, envDefault := range environmentDefaults {
// 		if os.Getenv(envVar) == "" {
// 			if err := s.Stager.WriteEnvFile(envVar, envDefault); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	if err := s.Stager.WriteEnvFile("NODE_HOME", filepath.Join(s.Stager.DepDir(), "node")); err != nil {
// 		return err
// 	}

// 	scriptContents := `export NODE_HOME=%s
// export NODE_ENV=${NODE_ENV:-production}
// export MEMORY_AVAILABLE=$(echo $VCAP_APPLICATION | jq '.limits.mem')
// export WEB_MEMORY=512
// export WEB_CONCURRENCY=1
// `

// 	return s.Stager.WriteProfileD("node.sh", fmt.Sprintf(scriptContents, filepath.Join("$DEPS_DIR", s.Stager.DepsIdx(), "node")))
// }
