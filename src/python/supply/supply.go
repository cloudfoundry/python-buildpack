package supply

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/python-buildpack/src/python/conda"
	"github.com/cloudfoundry/python-buildpack/src/python/pipfile"

	"os/exec"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/snapshot"
	"github.com/kr/text"
)

const EnvPipVersion = "BP_PIP_VERSION"

type Stager interface {
	BuildDir() string
	CacheDir() string
	DepDir() string
	DepsIdx() string
	LinkDirectoryInDepDir(destDir, destSubDir string) error
	WriteEnvFile(envVar, envVal string) error
	WriteProfileD(scriptName, scriptContents string) error
}

type Manifest interface {
	AllDependencyVersions(depName string) []string
	DefaultVersion(depName string) (libbuildpack.Dependency, error)
	IsCached() bool
}

type Installer interface {
	InstallDependency(dep libbuildpack.Dependency, outputDir string) error
	InstallOnlyVersion(depName, installDir string) error
}

type Command interface {
	Execute(dir string, stdout io.Writer, stderr io.Writer, program string, args ...string) error
	Output(dir string, program string, args ...string) (string, error)
	RunWithOutput(cmd *exec.Cmd) ([]byte, error)
}

type Supplier struct {
	PythonVersion          string
	Manifest               Manifest
	Installer              Installer
	Stager                 Stager
	Command                Command
	Log                    *libbuildpack.Logger
	Logfile                *os.File
	HasNltkData            bool
	removeRequirementsText bool
}

func Run(s *Supplier) error {
	if exists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "environment.yml")); err != nil {
		s.Log.Error("Error checking existence of environment.yml: %v", err)
		return err
	} else if exists {
		return conda.Run(conda.New(s.Installer, s.Stager, s.Command, s.Log))
	} else {
		return RunPython(s)
	}
}

func RunPython(s *Supplier) error {
	s.Log.BeginStep("Supplying Python")

	dirSnapshot := snapshot.Dir(s.Stager.BuildDir(), s.Log)
	if err := s.SetupCacheDir(); err != nil {
		s.Log.Error("Error setting up cache: %v", err)
		return err
	}

	if err := s.CopyRuntimeTxt(); err != nil {
		s.Log.Error("Error copying runtime.txt to deps dir: %v", err)
		return err
	}

	if err := s.HandlePipfile(); err != nil {
		s.Log.Error("Error checking for Pipfile.lock: %v", err)
		return err
	}

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

	if err := s.InstallPipEnv(); err != nil {
		s.Log.Error("Could not install pipenv: %v", err)
		return err
	}

	if err := s.HandleRequirementstxt(); err != nil {
		s.Log.Error("Error checking requirements.txt: %v", err)
		return err
	}

	if err := s.HandlePylibmc(); err != nil {
		s.Log.Error("Error checking Pylibmc: %v", err)
		return err
	}

	if err := s.HandleFfi(); err != nil {
		s.Log.Error("Error checking ffi: %v", err)
		return err
	}

	if err := s.HandleMercurial(); err != nil {
		s.Log.Error("Could not handle pip mercurial dependencies: %v", err)
		return err
	}

	if err := s.UninstallUnusedDependencies(); err != nil {
		s.Log.Error("Error uninstalling unused dependencies: %v", err)
		return err
	}

	vendored, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "vendor"))
	if err != nil {
		return fmt.Errorf("could not check vendor existence: %v", err)
	}

	if vendored {
		if err := s.RunPipVendored(); err != nil {
			s.Log.Error("Could not install vendored pip packages: %v", err)
			return err
		}
	} else {
		if err := s.RunPipUnvendored(); err != nil {
			s.Log.Error("Could not install pip packages: %v", err)
			return err
		}
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

	if cacheDirSize, err := s.Command.Output(os.Getenv("XDG_CACHE_HOME"), "du", "--summarize", os.Getenv("XDG_CACHE_HOME")); err == nil {
		s.Log.Debug("Size of pip cache dir: %s", cacheDirSize)
	}

	if s.removeRequirementsText {
		if err := os.Remove(filepath.Join(s.Stager.BuildDir(), "requirements.txt")); err != nil {
			s.Log.Error("Unable to clean up app directory: %s", err.Error())
			return err
		}
	}

	dirSnapshot.Diff()

	return nil
}

func (s *Supplier) CopyRuntimeTxt() error {
	if exists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "runtime.txt")); err != nil {
		return err
	} else if exists {
		if err = libbuildpack.CopyFile(filepath.Join(s.Stager.BuildDir(), "runtime.txt"), filepath.Join(s.Stager.DepDir(), "runtime.txt")); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supplier) HandleMercurial() error {
	if err := s.Command.Execute(s.Stager.BuildDir(), ioutil.Discard, ioutil.Discard, "grep", "-Fiq", "hg+", "requirements.txt"); err != nil {
		return nil
	}

	if s.Manifest.IsCached() {
		s.Log.Warning("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work.")
	}

	if err := s.runPipInstall("mercurial"); err != nil {
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin"); err != nil {
		return err
	}
	return nil
}

func (s *Supplier) HandlePipfile() error {
	var pipfileExists, runtimeExists bool
	var pipfileJson pipfile.Lock
	var err error

	if pipfileExists, err = libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "Pipfile.lock")); err != nil {
		return err
	}

	if runtimeExists, err = libbuildpack.FileExists(filepath.Join(s.Stager.DepDir(), "runtime.txt")); err != nil {
		return err
	}

	if pipfileExists && !runtimeExists {
		if err = libbuildpack.NewJSON().Load(filepath.Join(s.Stager.BuildDir(), "Pipfile.lock"), &pipfileJson); err != nil {
			return err
		}

		formattedVersion := s.formatVersion(pipfileJson.Meta.Requires.Version)

		if err := ioutil.WriteFile(filepath.Join(s.Stager.DepDir(), "runtime.txt"), []byte(formattedVersion), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Supplier) InstallPython() error {
	var dep libbuildpack.Dependency

	runtimetxtExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.DepDir(), "runtime.txt"))
	if err != nil {
		return err
	}

	if runtimetxtExists {
		userDefinedVersion, err := ioutil.ReadFile(filepath.Join(s.Stager.DepDir(), "runtime.txt"))
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
	if err := s.Installer.InstallDependency(dep, pythonInstallDir); err != nil {
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin"); err != nil {
		return err
	}
	if found, err := libbuildpack.FileExists(filepath.Join(pythonInstallDir, "usr", "lib", "x86_64-linux-gnu")); err != nil {
		return err
	} else if found {
		if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "usr", "lib", "x86_64-linux-gnu"), "lib"); err != nil {
			return err
		}
	}
	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib"); err != nil {
		return err
	}

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

func (s *Supplier) InstallPip() error {
	pipVersion := os.Getenv(EnvPipVersion)
	if pipVersion == "" {
		s.Log.Info("Using python's pip module")
		return nil
	}
	if pipVersion != "latest" {
		return fmt.Errorf("invalid pip version: %s", pipVersion)
	}

	tempPath := filepath.Join("/tmp", "pip")
	if err := s.Installer.InstallOnlyVersion("pip", tempPath); err != nil {
		return err
	}

	if err := s.Command.Execute(s.Stager.BuildDir(), indentWriter(os.Stdout), indentWriter(os.Stderr),
		"python",
		"-m", "pip",
		"install", "pip",
		"--exists-action=w",
		"--no-index",
		"--ignore-installed",
		fmt.Sprintf("--find-links=%s", tempPath),
	); err != nil {
		return err
	}

	return s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin")
}

func (s *Supplier) InstallPipPop() error {
	tempPath := filepath.Join("/tmp", "pip-pop")
	if err := s.Installer.InstallOnlyVersion("pip-pop", tempPath); err != nil {
		return err
	}

	if err := s.runPipInstall("pip-pop", "--exists-action=w", "--no-index", fmt.Sprintf("--find-links=%s", tempPath)); err != nil {
		return err
	}

	if err := s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin"); err != nil {
		return err
	}
	return nil
}

func (s *Supplier) InstallPipEnv() error {
	requirementstxtExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "requirements.txt"))
	if err != nil {
		return err
	} else if requirementstxtExists {
		return nil
	}

	pipfileExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "Pipfile"))
	if err != nil {
		return err
	} else if !pipfileExists {
		return nil
	}

	hasLockFile, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "Pipfile.lock"))
	if err != nil {
		return fmt.Errorf("could not check Pipfile.lock existence: %v", err)
	} else if hasLockFile {
		s.Log.Info("Generating 'requirements.txt' from Pipfile.lock")
		requirementsContents, err := pipfileToRequirements(filepath.Join(s.Stager.BuildDir(), "Pipfile.lock"))
		if err != nil {
			return fmt.Errorf("failed to write `requirement.txt` from Pipfile.lock: %s", err.Error())
		}

		return s.writeTempRequirementsTxt(requirementsContents)
	}

	s.Log.Info("Installing pipenv")
	if err := s.Installer.InstallOnlyVersion("pipenv", filepath.Join("/tmp", "pipenv")); err != nil {
		return err
	}

	if err := s.installFfi(); err != nil {
		return err
	}

	for _, dep := range []string{"setuptools_scm", "pytest-runner", "parver", "invoke", "pipenv", "wheel"} {
		s.Log.Info("Installing %s", dep)
		out := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		if err := s.runPipInstall(
			dep,
			"--exists-action=w",
			"--no-index",
			fmt.Sprintf("--find-links=%s", filepath.Join("/tmp", "pipenv")),
		); err != nil {
			return fmt.Errorf("Failed to install %s: %v.\nStdout: %v\nStderr: %v", dep, err, out, stderr)
		}
	}
	s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin")

	s.Log.Info("Generating 'requirements.txt' with pipenv")
	cmd := exec.Command("pipenv", "lock", "--requirements")
	cmd.Dir = s.Stager.BuildDir()
	cmd.Env = append(os.Environ(), "VIRTUALENV_NEVER_DOWNLOAD=true")
	output, err := s.Command.RunWithOutput(cmd)
	if err != nil {
		return err
	}
	outputString := string(output)

	// Remove output due to virtualenv
	if strings.HasPrefix(outputString, "Using ") {
		reqs := strings.SplitN(outputString, "\n", 2)
		if len(reqs) > 0 {
			outputString = reqs[1]
		}
	}

	return s.writeTempRequirementsTxt(outputString)
}

func pipfileToRequirements(lockFilePath string) (string, error) {
	var lockFile struct {
		Meta struct {
			Sources []struct {
				URL string
			}
		} `json:"_meta"`
		Default map[string]struct {
			Version string
		}
	}

	lockContents, err := ioutil.ReadFile(lockFilePath)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(lockContents, &lockFile)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}

	for i, source := range lockFile.Meta.Sources {
		if i == 0 {
			fmt.Fprintf(buf, "-i %s\n", source.URL)
		} else {
			fmt.Fprintf(buf, "--extra-index-url %s\n", source.URL)
		}
	}

	for pkg, obj := range lockFile.Default {
		fmt.Fprintf(buf, "%s%s\n", pkg, obj.Version)
	}

	return buf.String(), nil
}

func (s *Supplier) HandlePylibmc() error {
	memcachedDir := filepath.Join(s.Stager.DepDir(), "libmemcache")

	if err := s.Command.Execute(s.Stager.BuildDir(), ioutil.Discard, ioutil.Discard, "pip-grep", "-s", "requirements.txt", "pylibmc"); err == nil {
		s.Log.BeginStep("Noticed pylibmc. Bootstrapping libmemcached.")
		if err := s.Installer.InstallOnlyVersion("libmemcache", memcachedDir); err != nil {
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

func (s *Supplier) HandleRequirementstxt() error {
	if exists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "requirements.txt")); err != nil {
		return err
	} else if exists {
		return nil
	}

	if exists, err := libbuildpack.FileExists(filepath.Join(s.Stager.BuildDir(), "setup.py")); err != nil {
		return err
	} else if !exists {
		return nil
	}

	return s.writeTempRequirementsTxt("-e .")
}

func (s *Supplier) installFfi() error {
	ffiDir := filepath.Join(s.Stager.DepDir(), "libffi")

	// Only install libffi if we haven't done so already
	// This could be installed twice because pipenv installs it, but
	// we later run HandleFfi, which installs it if a dependency
	// from requirements.txt needs libffi.
	if os.Getenv("LIBFFI") != ffiDir {
		s.Log.BeginStep("Noticed dependency requiring libffi. Bootstrapping libffi.")
		if err := s.Installer.InstallOnlyVersion("libffi", ffiDir); err != nil {
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

func (s *Supplier) HandleFfi() error {
	if err := s.Command.Execute(s.Stager.BuildDir(), ioutil.Discard, ioutil.Discard, "pip-grep", "-s", "requirements.txt", "pymysql", "argon2-cffi", "bcrypt", "cffi", "cryptography", "django[argon2]", "Django[argon2]", "django[bcrypt]", "Django[bcrypt]", "PyNaCl", "pyOpenSSL", "PyOpenSSL", "requests[security]", "misaka"); err == nil {
		return s.installFfi()
	}
	return nil
}

func (s *Supplier) UninstallUnusedDependencies() error {
	requirementsDeclaredExists, err := libbuildpack.FileExists(filepath.Join(s.Stager.DepDir(), "python", "requirements-declared.txt"))
	if err != nil {
		return err
	}

	if requirementsDeclaredExists {
		fileContents, _ := ioutil.ReadFile(filepath.Join(s.Stager.DepDir(), "python", "requirements-declared.txt"))
		s.Log.Info("requirements-declared: %s", string(fileContents))

		staleContents, err := s.Command.Output(
			s.Stager.BuildDir(),
			"pip-diff",
			"--stale",
			filepath.Join(s.Stager.DepDir(), "python", "requirements-declared.txt"),
			filepath.Join(s.Stager.BuildDir(), "requirements.txt"),
			"--exclude",
			"setuptools",
			"pip",
			"wheel",
		)
		if err != nil {
			return err
		}

		if staleContents == "" {
			return nil
		}

		if err := ioutil.WriteFile(filepath.Join(s.Stager.DepDir(), "python", "requirements-stale.txt"), []byte(staleContents), 0644); err != nil {
			return err
		}

		s.Log.BeginStep("Uninstalling stale dependencies")
		if err := s.Command.Execute(
			s.Stager.BuildDir(),
			indentWriter(os.Stdout),
			indentWriter(os.Stderr),
			"python",
			"-m",
			"pip",
			"uninstall",
			"-r",
			filepath.Join(s.Stager.DepDir(), "python", "requirements-stale.txt", "-y", "--exists-action=w"),
		); err != nil {
			return err
		}

	}

	return nil
}

func (s *Supplier) RunPipUnvendored() error {
	shouldContinue, requirementsPath, err := s.shouldRunPip()
	if err != nil {
		return err
	} else if !shouldContinue {
		return nil
	}

	// Search lines from requirements.txt that begin with -i, --index-url, --extra-index-url or --trusted-host
	// and add them to the pydistutils file. We do this so that easy_install will use
	// the same indexes as pip. This may not actually be necessary because it's possible that
	// easy_install has been fixed upstream, but it has no ill side-effects.
	reqs, err := ioutil.ReadFile(requirementsPath)
	if err != nil {
		return fmt.Errorf("could not read requirements.txt: %v", err)
	}

	distUtils := map[string][]string{}

	re := regexp.MustCompile(`(?m)^\s*(-i|--index-url)\s+(.*)$`)
	match := re.FindStringSubmatch(string(reqs))
	if len(match) > 0 {
		distUtils["index_url"] = []string{match[len(match)-1]}
	}

	re = regexp.MustCompile(`(?m)^\s*--extra-index-url\s+(.*)$`)
	matches := re.FindAllStringSubmatch(string(reqs), -1)
	for _, m := range matches {
		distUtils["find_links"] = append(distUtils["find_links"], m[len(m)-1])
	}

	re = regexp.MustCompile(`(?m)^\s*--trusted-host\s+(.*)$`)
	matches = re.FindAllStringSubmatch(string(reqs), -1)
	if len(matches) > 0 {
		var allowHosts []string
		for _, m := range matches {
			allowHosts = append(allowHosts, m[len(m)-1])
		}
		distUtils["allow_hosts"] = []string{strings.Join(allowHosts, ",")}
	}

	if err := writePyDistUtils(distUtils); err != nil {
		return err
	}

	if err := s.runPipInstall(
		"-r", requirementsPath,
		"--ignore-installed",
		"--exists-action=w",
		"--src="+filepath.Join(s.Stager.DepDir(), "src"),
		"--disable-pip-version-check",
	); err != nil {
		return fmt.Errorf("could not run pip: %v", err)
	}

	return s.Stager.LinkDirectoryInDepDir(filepath.Join(s.Stager.DepDir(), "python", "bin"), "bin")
}

func (s *Supplier) RunPipVendored() error {
	shouldContinue, requirementsPath, err := s.shouldRunPip()
	if err != nil {
		return err
	} else if !shouldContinue {
		return nil
	}

	distUtils := map[string][]string{
		"allows_hosts": {""},
		"find_links":   {filepath.Join(s.Stager.BuildDir(), "vendor")},
	}
	if err := writePyDistUtils(distUtils); err != nil {
		return err
	}

	installArgs := []string{
		"-r", requirementsPath,
		"--ignore-installed",
		"--exists-action=w",
		"--src=" + filepath.Join(s.Stager.DepDir(), "src"),
		"--no-index",
		"--find-links=file://" + filepath.Join(s.Stager.BuildDir(), "vendor"),
		"--disable-pip-version-check",
	}

	if s.hasBuildOptions() {
		s.Log.Info("Using the pip --no-build-isolation flag since it is available")
		installArgs = append(installArgs, "--no-build-isolation")
	}

	// Remove lines from requirements.txt that begin with -i
	// because specifying index links here makes pip always want internet access,
	// and pipenv generates requirements.txt with -i.
	originalReqs, err := ioutil.ReadFile(requirementsPath)
	if err != nil {
		return fmt.Errorf("could not read requirements.txt: %v", err)
	}

	re := regexp.MustCompile(`(?m)^\s*-i.*$`)
	modifiedReqs := re.ReplaceAll(originalReqs, []byte{})
	err = ioutil.WriteFile(requirementsPath, modifiedReqs, 0644)
	if err != nil {
		return fmt.Errorf("could not overwrite requirements file: %v", err)
	}

	if err := s.runPipInstall(installArgs...); err != nil {
		s.Log.Info("Running pip install without indexes failed. Not all dependencies were vendored. Trying again with indexes.")

		if err := ioutil.WriteFile(requirementsPath, originalReqs, 0644); err != nil {
			return fmt.Errorf("could not overwrite modified requirements file: %v", err)
		}

		if err := s.RunPipUnvendored(); err != nil {
			s.Log.Info("Running pip install failed. You need to include all dependencies in the vendor directory.")
			return err
		}
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
export FORWARDED_ALLOW_IPS='*'
export GUNICORN_CMD_ARGS=${GUNICORN_CMD_ARGS:-'--access-logfile -'}
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

	if err := s.Command.Execute("/", indentWriter(os.Stdout), indentWriter(os.Stderr), "python", args...); err != nil {
		return err
	}

	s.HasNltkData = true

	return nil
}

func (s *Supplier) SetupCacheDir() error {
	if err := os.Setenv("XDG_CACHE_HOME", filepath.Join(s.Stager.CacheDir(), "pip_cache")); err != nil {
		return err
	}
	if err := s.Stager.WriteEnvFile("XDG_CACHE_HOME", filepath.Join(s.Stager.CacheDir(), "pip_cache")); err != nil {
		return err
	}
	return nil
}

func writePyDistUtils(distUtils map[string][]string) error {
	pyDistUtilsPath := filepath.Join(os.Getenv("HOME"), ".pydistutils.cfg")

	b := strings.Builder{}
	b.WriteString("[easy_install]\n")
	for k, v := range distUtils {
		b.WriteString(fmt.Sprintf("%s = %s\n", k, strings.Join(v, "\n\t")))
	}

	if err := ioutil.WriteFile(pyDistUtilsPath, []byte(b.String()), os.ModePerm); err != nil {
		return err
	}

	return nil
}

func (s *Supplier) shouldRunPip() (bool, string, error) {
	s.Log.BeginStep("Running Pip Install")
	if os.Getenv("PIP_CERT") == "" {
		os.Setenv("PIP_CERT", "/etc/ssl/certs/ca-certificates.crt")
	}

	requirementsPath := filepath.Join(s.Stager.BuildDir(), "requirements.txt")
	if exists, err := libbuildpack.FileExists(requirementsPath); err != nil {
		return false, "", fmt.Errorf("could not determine existence of requirements.txt: %v", err)
	} else if !exists {
		s.Log.Debug("Skipping 'pip install' since requirements.txt does not exist")
		return false, "", nil
	}

	return true, requirementsPath, nil
}

func pipCommand() []string {
	if os.Getenv(EnvPipVersion) != "" {
		return []string{"pip"}
	}
	return []string{"python", "-m", "pip"}
}

func (s *Supplier) runPipInstall(args ...string) error {
	installCmd := append(append(pipCommand(), "install"), args...)
	return s.Command.Execute(s.Stager.BuildDir(), indentWriter(os.Stdout), indentWriter(os.Stderr), installCmd[0], installCmd[1:]...)
}

func (s *Supplier) formatVersion(version string) string {
	verSlice := strings.Split(version, ".")

	if len(verSlice) < 3 {
		return fmt.Sprintf("python-%s.x", version)
	}

	return fmt.Sprintf("python-%s", version)

}

func (s *Supplier) writeTempRequirementsTxt(content string) error {
	s.removeRequirementsText = true
	return ioutil.WriteFile(filepath.Join(s.Stager.BuildDir(), "requirements.txt"), []byte(content), 0644)
}

func (s *Supplier) hasBuildOptions() bool {
	helpCommand := append(pipCommand(), "install", "--no-build-isolation", "-h")
	err := s.Command.Execute(s.Stager.BuildDir(), nil, nil, helpCommand[0], helpCommand[1:]...)
	return nil == err
}

func indentWriter(writer io.Writer) io.Writer {
	return text.NewIndentWriter(writer, []byte("       "))
}
