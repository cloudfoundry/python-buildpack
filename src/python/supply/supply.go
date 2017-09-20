package supply

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	BuildDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(string, string) error
	WriteEnvFile(string, string) error
	WriteProfileD(string, string) error
	// SetStagingEnvironment() error
}

type Manifest interface {
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
	IsCached() bool
}

type Command interface {
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
	// Run(cmd *exec.Cmd) error
}

type Supplier struct {
	PythonVersion string
	Manifest      Manifest
	Stager        Stager
	Command       Command
	Log           *libbuildpack.Logger
	Logfile       *os.File
	HasNltkData   bool
}

func Run(s *Supplier) error {
	// FIXME handle errors

	// TODO: Procfile warning ?

	// TODO: Restore cache?

	// TODO: handle vv c.f. supply bash script
	// # # Pipenv Python version support.
	// # $BIN_DIR/steps/pipenv-python-version.rb $BUILD_DIR
	if err := s.InstallPython(); err != nil {
		s.Log.Error("Could not install python: %v", err)
		return err
	}

	if err := s.InstallPip(); err != nil {
		s.Log.Error("Could not install pip: %v", err)
		return err
	}

	if err := s.InstallPipPop(); err != nil {
		s.Log.Error("Could not install pip pop: %v", err)
		return err
	}

	// TODO: pipenv ?

	if err := s.HandlePylibmc(); err != nil {
		s.Log.Error("Error checking Pylibmc: %v", err)
		return err
	}

	// TODO: handle vv c.f. supply bash script
	// # # Automatic configuration for Gunicorn's ForwardedAllowIPS setting.
	// # echo "export FORWARDED_ALLOW_IPS='*'" > $DEPS_DIR/$DEPS_IDX/profile.d/python.gunicorn.sh

	// TODO: handle vv c.f. supply bash script
	// # # If no requirements.txt file given, assume `setup.py develop` is intended.
	// # if [ ! -f requirements.txt ]; then
	// #   echo "-e ." > requirements.txt
	// # fi
	if err := s.HandleFfi(); err != nil {
		s.Log.Error("Error checking ffi: %v", err)
		return err
	}

	// TODO: write config.yml

	// TODO: build PATH and LD_LIBRARY_PATH ? dont think this is necessary?

	// TODO: steps/mercurial ?
	if err := s.HandleMercurial(); err != nil {
		s.Log.Error("Could not handle pip mercurial dependencies: %v", err)
		return err
	}

	// TODO: steps/pip-uninstall ?

	if err := s.RunPip(); err != nil {
		s.Log.Error("Could not install pip packages: %v", err)
		return err
	}

	if err := s.DownloadNLTKCorpora(); err != nil {
		s.Log.Error("Could not download NLTK Corpora: %v", err)
		return err
	}

	if err := s.RewriteShebangs(); err != nil {
		s.Log.Error("Unable to rewrite she-bangs: %s", err.Error())
		return err
	}

	if err := s.CreateDefaultEnv(); err != nil {
		s.Log.Error("Unable to setup default environment: %s", err.Error())
		return err
	}

	// TODO: caching?

	return nil
}

func (s *Supplier) HandleMercurial() error {
	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "grep", "-Fiq", "hg+", "requirements.txt"); err != nil {
		return nil
	}

	if s.Manifest.IsCached() {
		s.Log.Warning("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work.")
	}

	//TODO: output pipe through equivalent of shell: cleanup | indent
	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip", "install", "mercurial"); err != nil {
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin"); err != nil {
		return err
	}
	return nil
}

func (s *Supplier) InstallPython() error {
	var dep libbuildpack.Dependency

	runtimetxtExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "runtime.txt"))
	if err != nil {
		return err
	}

	if runtimetxtExists {
		userDefinedVersion, err := ioutil.ReadFile(filepath.Join(s.Stager.BuildDir(), "runtime.txt"))
		if err != nil {
			return err
		}

		s.PythonVersion = strings.TrimSpace(strings.NewReplacer("\\r", "", "\\n", "").Replace(string(userDefinedVersion)))
		s.Log.Debug("***Version info: (%s)", s.PythonVersion)
	}

	if s.PythonVersion != "" {
		versions := s.Manifest.AllDependencyVersions("python")
		shortPythonVersion := strings.TrimLeft(s.PythonVersion, "python-")
		s.Log.Debug("***Version info: (%s) (%s)", s.PythonVersion, shortPythonVersion)
		ver, err := libbuildpack.FindMatchingVersion(shortPythonVersion, versions)
		if err != nil {
			return err
		}
		dep.Name = "python"
		dep.Version = ver
		s.Log.Debug("***Version info: %s, %s, %s", dep.Name, s.PythonVersion, dep.Version)
	} else {
		var err error

		dep, err = s.Manifest.DefaultVersion("python")
		if err != nil {
			return err
		}
	}

	pythonInstallDir := filepath.Join(s.Stager.DepDir(), "python")
	if err := s.Manifest.InstallDependency(dep, pythonInstallDir); err != nil {
		return err
	}

	s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
	s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")

	if err := os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Join(s.Stager.DepDir(), "bin"), os.Getenv("PATH"))); err != nil {
		return err
	}
	if err := os.Setenv("PYTHONPATH", filepath.Join(s.Stager.DepDir())); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) RewriteShebangs() error {
	files, err := filepath.Glob(filepath.Join(s.Stager.DepDir(), "bin", "*"))
	if err != nil {
		return err
	}

	for _, file := range files {
		if fileInfo, err := os.Stat(file); err != nil {
			return err
		} else if fileInfo.IsDir() {
			continue
		}
		fileContents, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		shebangRegex := regexp.MustCompile(`^#!/.*/python.*`)
		fileContents = shebangRegex.ReplaceAll(fileContents, []byte("#!/usr/bin/env python"))
		if err := ioutil.WriteFile(file, fileContents, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supplier) InstallPipPop() error {
	tempPath := filepath.Join("/tmp", "pip-pop")
	if err := s.Manifest.InstallOnlyVersion("pip-pop", tempPath); err != nil {
		return err
	}

	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip", "install", "pip-pop", "--exists-action=w", "--no-index", fmt.Sprintf("--find-links=%s", tempPath)); err != nil {
		s.Log.Debug("******Path val: %s", os.Getenv("PATH"))
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin"); err != nil {
		return err
	}
	return nil
}

func (s *Supplier) HandlePylibmc() error {
	memcachedDir := filepath.Join(s.Stager.DepDir(), "libmemcache")
	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip-grep", "-s", "requirements.txt", "pylibmc"); err == nil {
		s.Log.BeginStep("Noticed pylibmc. Bootstrapping libmemcached.")
		if err := s.Manifest.InstallOnlyVersion("libmemcache", memcachedDir); err != nil {
			return err
		}
		os.Setenv("LIBMEMCACHED", memcachedDir)
		s.Stager.WriteEnvFile("LIBMEMCACHED", memcachedDir)
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib"), "lib")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "sasl2"), "lib")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "pkgconfig"), "pkgconfig")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(memcachedDir, "include"), "include")
	}

	return nil
}

func (s *Supplier) HandleFfi() error {
	ffiDir := filepath.Join(s.Stager.DepDir(), "libffi")
	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip-grep", "-s", "requirements.txt", "argon2-cffi", "bcrypt", "cffi", "cryptography", "django[argon2]", "Django[argon2]", "django[bcrypt]", "Django[bcrypt]", "PyNaCl", "pyOpenSSL", "PyOpenSSL", "requests[security]", "misaka"); err == nil {
		s.Log.BeginStep("Noticed dependency requiring libffi. Bootstrapping libffi.")
		if err := s.Manifest.InstallOnlyVersion("libffi", ffiDir); err != nil {
			return err
		}
		versions := s.Manifest.AllDependencyVersions("libffi")
		os.Setenv("LIBFFI", ffiDir)
		s.Stager.WriteEnvFile("LIBFFI", ffiDir)
		s.Stager.LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib"), "lib")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib", "pkgconfig"), "pkgconfig")
		s.Stager.LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib", "libffi-"+versions[0], "include"), "include")
	}

	return nil
}

func (s *Supplier) InstallPip() error {
	for _, name := range []string{"setuptools", "pip"} {
		if err := s.Manifest.InstallOnlyVersion(name, filepath.Join("/tmp", name)); err != nil {
			return err
		}
		versions := s.Manifest.AllDependencyVersions(name)
		if output, err := s.Command.Output(filepath.Join("/tmp", name, name+"-"+versions[0]), "python", "setup.py", "install", fmt.Sprintf("--prefix=%s", filepath.Join(s.Stager.DepDir(), "python"))); err != nil {
			s.Log.Error(output)
			return err
		}
	}

	for _, dir := range []string{"bin", "lib", "include"} {
		if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", dir), dir); err != nil {
			return err
		}
	}
	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "lib", "pkgconfig"), "pkgconfig"); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) RunPip() error {
	installArgs := []string{"install", "-r", "requirements.txt", "--exists-action=w", "--src=" + filepath.Join(s.Stager.DepDir(), "src")}
	if vendorExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "vendor")); err != nil {
		return fmt.Errorf("Couldn't check vendor existence: %v", err)
	} else if vendorExists {
		installArgs = append(installArgs, "--no-index", "--find-links=file://"+filepath.Join(s.Stager.BuildDir(), "vendor"))
	}

	if err := s.Command.Execute(s.Stager.BuildDir(), os.Stdout, os.Stderr, "pip", installArgs...); err != nil {
		s.Log.Debug("******Path val: %s", os.Getenv("PATH"))
		return err
	}

	return s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin")
}

func (s *Supplier) CreateDefaultEnv() error {
	var environmentVars = map[string]string{
		"PYTHONPATH":       s.Stager.DepDir(),
		"LIBRARY_PATH":     filepath.Join(s.Stager.DepDir(), "lib"),
		"PYTHONHOME":       filepath.Join(s.Stager.DepDir(), "python"),
		"PYTHONUNBUFFERED": "1",
		"PYTHONHASHSEED":   "random",
		"LANG":             "en_US.UTF-8",
	}

	scriptContents := fmt.Sprintf(`export LANG=${LANG:-en_US.UTF-8}
export PYTHONHASHSEED=${PYTHONHASHSEED:-random}
export PYTHONPATH=$DEPS_DIR/%s
export PYTHONHOME=$DEPS_DIR/%s/python
export PYTHONUNBUFFERED=1
`, s.Stager.DepsIdx(), s.Stager.DepsIdx())

	if s.HasNltkData {
		scriptContents += fmt.Sprintf(`export NLTK_DATA=$DEPS_DIR/%s/python/nltk_data`, s.Stager.DepsIdx())
		environmentVars["NLTK_DATA"] = filepath.Join(s.Stager.DepDir(), "python", "nltk_data")
	}

	for envVar, envValue := range environmentVars {
		if err := s.Stager.WriteEnvFile(envVar, envValue); err != nil {
			return err
		}
	}

	return s.Stager.WriteProfileD("python.sh", scriptContents)
}

func (s *Supplier) DownloadNLTKCorpora() error {
	if err := s.Command.Execute("/", ioutil.Discard, ioutil.Discard, "python", "-m", "nltk.downloader", "-h"); err != nil {
		return nil
	}

	s.Log.BeginStep("Downloading NLTK corpora...")

	if exists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "nltk.txt")); err != nil {
		return fmt.Errorf("Couldn't check nltk.txt existence: %v", err)
	} else if !exists {
		s.Log.Info("nltk.txt not found, not downloading any corpora")
		return nil
	}

	bPackages, err := ioutil.ReadFile(filepath.Join(s.Stager.BuildDir(), "nltk.txt"))
	if err != nil {
		return err
	}
	sPackages := strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(string(bPackages)))
	args := []string{"-m", "nltk.downloader", "-d", filepath.Join(s.Stager.DepDir(), "python", "nltk_data")}
	args = append(args, strings.Split(sPackages, " ")...)

	s.Log.BeginStep("Downloading NLTK packages: %s", sPackages)

	if err := s.Command.Execute("/", os.Stdout, os.Stderr, "python", args...); err != nil {
		return err
	}

	s.HasNltkData = true

	return nil
}
