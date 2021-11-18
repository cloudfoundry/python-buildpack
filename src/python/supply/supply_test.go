package supply_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/python-buildpack/src/python/supply"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		err           error
		buildDir      string
		cacheDir      string
		depsDir       string
		depsIdx       string
		depDir        string
		supplier      *supply.Supplier
		logger        *libbuildpack.Logger
		buffer        *bytes.Buffer
		mockCtrl      *gomock.Controller
		mockManifest  *MockManifest
		mockInstaller *MockInstaller
		mockStager    *MockStager
		mockCommand   *MockCommand
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "python-buildpack.cache.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "python-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "13"

		depDir = filepath.Join(depsDir, depsIdx)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockInstaller = NewMockInstaller(mockCtrl)
		mockStager = NewMockStager(mockCtrl)
		mockStager.EXPECT().BuildDir().AnyTimes().Return(buildDir)
		mockStager.EXPECT().CacheDir().AnyTimes().Return(cacheDir)
		mockStager.EXPECT().DepDir().AnyTimes().Return(depDir)
		mockStager.EXPECT().DepsIdx().AnyTimes().Return(depsIdx)
		mockCommand = NewMockCommand(mockCtrl)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		supplier = &supply.Supplier{
			Manifest:  mockManifest,
			Installer: mockInstaller,
			Stager:    mockStager,
			Command:   mockCommand,
			Log:       logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())
	})

	Describe("InstallPython", func() {
		var pythonInstallDir string
		var versions []string
		var originalPath string

		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
			pythonInstallDir = filepath.Join(depDir, "python")
			Expect(ioutil.WriteFile(filepath.Join(depDir, "runtime.txt"), []byte("\n\n\npython-3.4.2\n\n\n"), 0644)).To(Succeed())

			versions = []string{"3.4.2"}
			originalPath = os.Getenv("PATH")
		})

		AfterEach(func() {
			os.Setenv("PATH", originalPath)
		})

		Context("runtime.txt sets Python version 3", func() {
			It("installs Python version 3", func() {
				mockManifest.EXPECT().AllDependencyVersions("python").Return(versions)
				mockInstaller.EXPECT().InstallDependency(libbuildpack.Dependency{Name: "python", Version: "3.4.2"}, pythonInstallDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")
				Expect(supplier.InstallPython()).To(Succeed())
				Expect(os.Getenv("PATH")).To(Equal(fmt.Sprintf("%s:%s", filepath.Join(depDir, "bin"), originalPath)))
				Expect(os.Getenv("PYTHONPATH")).To(Equal(filepath.Join(depDir)))
			})
		})

		Context("no runtime.txt is provided", func() {
			BeforeEach(func() {
				Expect(os.RemoveAll(filepath.Join(depDir, "runtime.txt"))).To(Succeed())
			})

			It("installs the default Python version", func() {
				mockManifest.EXPECT().DefaultVersion("python").Return(libbuildpack.Dependency{Name: "python", Version: "some-default-version"}, nil)
				mockInstaller.EXPECT().InstallDependency(libbuildpack.Dependency{Name: "python", Version: "some-default-version"}, pythonInstallDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "bin"), "bin")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(pythonInstallDir, "lib"), "lib")
				Expect(supplier.InstallPython()).To(Succeed())
				Expect(os.Getenv("PATH")).To(Equal(fmt.Sprintf("%s:%s", filepath.Join(depDir, "bin"), originalPath)))
				Expect(os.Getenv("PYTHONPATH")).To(Equal(filepath.Join(depDir)))
			})
		})
	})

	Describe("InstallPip", func() {
		Describe("BP_PIP_VERSION not set", func() {
			BeforeEach(func() {
				Expect(os.Unsetenv("BP_PIP_VERSION")).To(Succeed())
			})

			It("skips install", func() {
				mockInstaller.EXPECT().InstallOnlyVersion(gomock.Any(), gomock.Any()).Times(0)
				mockCommand.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				mockStager.EXPECT().LinkDirectoryInDepDir(gomock.Any(), gomock.Any()).Times(0)

				Expect(supplier.InstallPip()).To(Succeed())
			})

			It("uses python's pip module for installs", func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte{}, 0644)).To(Succeed())
				mockStager.EXPECT().LinkDirectoryInDepDir(gomock.Any(), gomock.Any())

				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--disable-pip-version-check")
				Expect(supplier.RunPipUnvendored()).To(Succeed())
			})
		})

		Describe("BP_PIP_VERSION set to 'latest'", func() {
			BeforeEach(func() {
				Expect(os.Setenv("BP_PIP_VERSION", "latest")).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Unsetenv("BP_PIP_VERSION")).To(Succeed())
			})

			It("installs latest from manifest", func() {
				mockInstaller.EXPECT().InstallOnlyVersion("pip", "/tmp/pip")
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "pip", "--exists-action=w", "--no-index", "--ignore-installed", "--find-links=/tmp/pip")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(filepath.Join(depDir, "python"), "bin"), "bin")

				Expect(supplier.InstallPip()).To(Succeed())
			})

			It("uses installed pip for installs", func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte{}, 0644)).To(Succeed())
				mockStager.EXPECT().LinkDirectoryInDepDir(gomock.Any(), gomock.Any())

				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--disable-pip-version-check")
				Expect(supplier.RunPipUnvendored()).To(Succeed())
			})
		})

		Describe("BP_PIP_VERSION is invalid", func() {
			BeforeEach(func() {
				Expect(os.Setenv("BP_PIP_VERSION", "something-else")).To(Succeed())
			})

			AfterEach(func() {
				Expect(os.Unsetenv("BP_PIP_VERSION")).To(Succeed())
			})

			It("returns an error without installing", func() {
				mockInstaller.EXPECT().InstallOnlyVersion(gomock.Any(), gomock.Any()).Times(0)
				mockCommand.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				mockStager.EXPECT().LinkDirectoryInDepDir(gomock.Any(), gomock.Any()).Times(0)

				Expect(supplier.InstallPip()).To(MatchError("invalid pip version: something-else"))
			})
		})
	})

	Describe("HandlePipfile", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
			pipfileContents := `
			{
				"_meta":{
					"requires":{
						"python_version":"3.6"
					}
				}
			}`

			Expect(ioutil.WriteFile(filepath.Join(buildDir, "Pipfile.lock"), []byte(pipfileContents), 0644)).To(Succeed())
		})

		It("creates runtime.txt from Pipfile.lock contents if none exists", func() {
			Expect(supplier.HandlePipfile()).To(Succeed())
			runtimeContents, err := ioutil.ReadFile(filepath.Join(depDir, "runtime.txt"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(runtimeContents)).To(ContainSubstring("python-3.6"))
		})
	})

	Describe("InstallPipPop", func() {
		It("installs pip-pop", func() {
			mockInstaller.EXPECT().InstallOnlyVersion("pip-pop", "/tmp/pip-pop")
			mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "pip-pop", "--exists-action=w", "--no-index", "--find-links=/tmp/pip-pop")
			mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(filepath.Join(depDir, "python"), "bin"), "bin")
			Expect(supplier.InstallPipPop()).To(Succeed())
		})
	})

	// Add the expects for what the installFfi function uses
	expectInstallFfi := func() string {
		ffiDir := filepath.Join(depDir, "libffi")
		mockManifest.EXPECT().AllDependencyVersions("libffi").Return([]string{"1.2.3"})
		mockInstaller.EXPECT().InstallOnlyVersion("libffi", ffiDir)
		mockStager.EXPECT().WriteEnvFile("LIBFFI", ffiDir)
		mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib"), "lib")
		mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib", "pkgconfig"), "pkgconfig")
		mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(ffiDir, "lib", "libffi-1.2.3", "include"), "include")
		return ffiDir
	}

	// Add the expects for functions used to install pipenv
	// returns ffidir for convenience
	expectInstallPipEnv := func() string {
		// install pipenv binary from bp manifest
		mockInstaller.EXPECT().InstallOnlyVersion("pipenv", "/tmp/pipenv")

		// install pipenv dependencies
		ffiDir := expectInstallFfi()
		mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "setuptools_scm", "--exists-action=w", "--no-index", "--find-links=/tmp/pipenv")
		mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "pytest-runner", "--exists-action=w", "--no-index", "--find-links=/tmp/pipenv")
		mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "parver", "--exists-action=w", "--no-index", "--find-links=/tmp/pipenv")
		mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "invoke", "--exists-action=w", "--no-index", "--find-links=/tmp/pipenv")
		mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "pipenv", "--exists-action=w", "--no-index", "--find-links=/tmp/pipenv")
		mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "wheel", "--exists-action=w", "--no-index", "--find-links=/tmp/pipenv")

		mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(filepath.Join(depDir, "python"), "bin"), "bin")
		return ffiDir
	}

	Describe("InstallPipEnv", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
		})

		Context("when Pipfile.lock and requirements.txt both exist", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "Pipfile.lock"), []byte("This is pipfile"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte("blah"), 0644)).To(Succeed())
			})

			It("does not install Pipenv", func() {
				Expect(supplier.InstallPipEnv()).To(Succeed())
			})
		})

		Context("when Pipfile.lock exists but requirements.txt does not exist", func() {
			BeforeEach(func() {
				const lockFileContnet string = `{"_meta":{"sources":[{"url":"https://pypi.org/simple"},{"url":"https://pypi.example.org/simple"}]},"default":{"test":{"version":"==1.2.3"}}}`
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "Pipfile"), []byte("some content"), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "Pipfile.lock"), []byte(lockFileContnet), 0644)).To(Succeed())
			})

			It("manually generates the requirements.txt", func() {
				Expect(supplier.InstallPipEnv()).To(Succeed())

				requirementsContents, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(requirementsContents).To(ContainSubstring("-i https://pypi.org/simple"))
				Expect(requirementsContents).To(ContainSubstring("--extra-index-url https://pypi.example.org/simple"))
				Expect(requirementsContents).To(ContainSubstring("test==1.2.3"))
			})
		})

		Context("when Pipfile exists but requirements.txt and Pipfile.lock do not exist", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "Pipfile"), []byte("some content"), 0644)).To(Succeed())
			})

			It("manually generates the requirements.txt", func() {
				expectInstallPipEnv()

				mockCommand.EXPECT().RunWithOutput(gomock.Any()).Return([]byte("Using /tmp/deps/0/bin/python3.6m to create virtualenv…\nline 1\nline 2\n"), nil)

				Expect(supplier.InstallPipEnv()).To(Succeed())

				requirementsContents, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
				Expect(err).ToNot(HaveOccurred())

				By("removes extraneous pipenv lock output")
				Expect(string(requirementsContents)).To(Equal("line 1\nline 2\n"))
			})
		})
	})

	Describe("HandlePylibmc", func() {
		AfterEach(func() {
			os.Setenv("LIBMEMCACHED", "")
		})

		Context("when the app uses pylibmc", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "pylibmc").Return(nil)
			})
			It("installs libmemcache", func() {
				memcachedDir := filepath.Join(depDir, "libmemcache")
				mockInstaller.EXPECT().InstallOnlyVersion("libmemcache", memcachedDir)
				mockStager.EXPECT().WriteEnvFile("LIBMEMCACHED", memcachedDir)
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib"), "lib")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "sasl2"), "lib")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "lib", "pkgconfig"), "pkgconfig")
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(memcachedDir, "include"), "include")
				Expect(supplier.HandlePylibmc()).To(Succeed())
				Expect(os.Getenv("LIBMEMCACHED")).To(Equal(memcachedDir))
			})
		})
		Context("when the app does not use pylibmc", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "pylibmc").Return(fmt.Errorf("not found"))
			})

			It("does not install libmemcache", func() {
				Expect(supplier.HandlePylibmc()).To(Succeed())
				Expect(os.Getenv("LIBMEMCACHED")).To(Equal(""))
			})
		})
	})

	Describe("CopyRuntimeTxt", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
		})

		It("succeeds without requirements.txt and runtime.txt in build dir", func() {
			Expect(supplier.CopyRuntimeTxt()).To(Succeed())
		})

		Context("requirements.txt and runtime.txt in build dir", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "runtime.txt"), []byte("blah blah"), 0644)).To(Succeed())
			})

			It("copies requirements.txt and runtime.txt", func() {
				Expect(supplier.CopyRuntimeTxt()).To(Succeed())

				fileContents, err := ioutil.ReadFile(filepath.Join(depDir, "runtime.txt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(fileContents).To(Equal([]byte("blah blah")))
			})
		})
	})

	Describe("HandleRequirementstxt", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
		})
		Context("when requirements.txt does not exist", func() {
			Context("when setup.py exists", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(buildDir, "setup.py"), []byte{}, 0644)).To(Succeed())
				})
				It("create requirements.txt with '-e .'", func() {
					Expect(supplier.HandleRequirementstxt()).To(Succeed())

					Expect(filepath.Join(buildDir, "requirements.txt")).To(BeARegularFile())

					fileContents, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
					Expect(err).ToNot(HaveOccurred())
					Expect(fileContents).To(Equal([]byte("-e .")))
				})
			})
			Context("when setup.py does not exist", func() {
				It("does not create requirements.txt file", func() {
					Expect(supplier.HandleRequirementstxt()).To(Succeed())
					Expect(filepath.Join(buildDir, "requirements.txt")).ToNot(BeARegularFile())
				})
			})
		})

		Context("when requirements.txt exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte("blah"), 0644)).To(Succeed())
			})

			It("does nothing", func() {
				Expect(supplier.HandleRequirementstxt()).To(Succeed())
				fileContents, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(fileContents).To(Equal([]byte("blah")))
			})
		})
	})

	Describe("HandleFfi", func() {
		AfterEach(func() {
			os.Setenv("LIBFFI", "")
		})

		Context("when the app uses ffi", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "pymysql", "argon2-cffi", "bcrypt", "cffi", "cryptography", "django[argon2]", "Django[argon2]", "django[bcrypt]", "Django[bcrypt]", "PyNaCl", "pyOpenSSL", "PyOpenSSL", "requests[security]", "misaka").Return(nil)
			})

			It("installs ffi", func() {
				ffiDir := expectInstallFfi()
				Expect(supplier.HandleFfi()).To(Succeed())
				Expect(os.Getenv("LIBFFI")).To(Equal(ffiDir))
			})

			Context("when pipenv is installed", func() {
				var ffiDir string
				BeforeEach(func() {
					// expect pipenv to be installed, and for it to install ffi
					Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(buildDir, "Pipfile"), []byte("This is pipfile"), 0644)).To(Succeed())
					ffiDir = expectInstallPipEnv()
					mockCommand.EXPECT().RunWithOutput(gomock.Any()).Return([]byte("test"), nil)

					// install pipenv
					Expect(supplier.InstallPipEnv()).To(Succeed())
				})
				It("it doesn't install ffi a second time", func() {
					Expect(supplier.HandleFfi()).To(Succeed())
					Expect(os.Getenv("LIBFFI")).To(Equal(ffiDir))
				})
			})
		})
		Context("when the app does not use libffi", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "pip-grep", "-s", "requirements.txt", "pymysql", "argon2-cffi", "bcrypt", "cffi", "cryptography", "django[argon2]", "Django[argon2]", "django[bcrypt]", "Django[bcrypt]", "PyNaCl", "pyOpenSSL", "PyOpenSSL", "requests[security]", "misaka").Return(fmt.Errorf("not found"))
			})

			It("does not install libffi", func() {
				Expect(supplier.HandleFfi()).To(Succeed())
				Expect(os.Getenv("LIBFFI")).To(Equal(""))
			})
		})
	})

	Describe("HandleMercurial", func() {
		Context("has mercurial dependencies", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "grep", "-Fiq", "hg+", "requirements.txt")
			})

			Context("the buildpack is not cached", func() {
				BeforeEach(func() {
					mockManifest.EXPECT().IsCached().Return(false)
				})
				It("installs mercurial", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "mercurial")
					mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "python", "bin"), "bin")
					Expect(supplier.HandleMercurial()).To(Succeed())
				})
			})

			Context("the buildpack is cached", func() {
				BeforeEach(func() {
					mockManifest.EXPECT().IsCached().Return(true)
				})
				It("installs mercurial and provides a warning", func() {
					mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "mercurial")
					mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "python", "bin"), "bin")
					Expect(supplier.HandleMercurial()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("Cloud Foundry does not support Pip Mercurial dependencies while in offline-mode. Vendor your dependencies if they do not work."))
				})
			})

		})
		Context("does not have mercurial dependencies", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "grep", "-Fiq", "hg+", "requirements.txt").Return(fmt.Errorf("Mercurial not found"))
			})

			It("succeeds without installing mercurial", func() {
				Expect(supplier.HandleMercurial()).To(Succeed())
			})
		})
	})

	Describe("RewriteShebangs", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(depDir, "bin"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(depDir, "bin", "somescript"), []byte("#!/usr/bin/python\n\n\n"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(depDir, "bin", "anotherscript"), []byte("#!//bin/python\n\n\n"), 0755)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(depDir, "bin", "__pycache__"), 0755)).To(Succeed())
			Expect(os.Symlink(filepath.Join(depDir, "bin", "__pycache__"), filepath.Join(depDir, "bin", "__pycache__SYMLINK"))).To(Succeed())
		})
		It("changes them to #!/usr/bin/env python", func() {
			Expect(supplier.RewriteShebangs()).To(Succeed())

			fileContents, err := ioutil.ReadFile(filepath.Join(depDir, "bin", "somescript"))
			Expect(err).ToNot(HaveOccurred())

			secondFileContents, err := ioutil.ReadFile(filepath.Join(depDir, "bin", "anotherscript"))
			Expect(err).ToNot(HaveOccurred())

			Expect(string(fileContents)).To(HavePrefix("#!/usr/bin/env python"))
			Expect(string(secondFileContents)).To(HavePrefix("#!/usr/bin/env python"))
		})
	})

	Describe("UninstallUnusedDependencies", func() {
		Context("when requirements-declared.txt exists", func() {

			requirementsDeclared :=
				`Flask==0.10.1
Jinja2==2.7.2
MarkupSafe==0.21
Werkzeug==0.10.4
gunicorn==19.3.0
itsdangerous==0.24
pylibmc==1.4.2
cffi==0.9.2
`
			requirements :=
				`Flask==0.10.1
Jinja2==2.7.2
MarkupSafe==0.21
`
			requirementsStale :=
				`Werkzeug==0.10.4
gunicorn==19.3.0
itsdangerous==0.24
pylibmc==1.4.2
cffi==0.9.2
`
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(depDir, "python"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(depDir, "python", "requirements-declared.txt"), []byte(requirementsDeclared), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte(requirements), 0644)).To(Succeed())
			})

			It("creates requirements-stale.txt and uninstalls unused dependencies", func() {
				mockCommand.EXPECT().Output(buildDir, "pip-diff", "--stale", filepath.Join(depDir, "python", "requirements-declared.txt"), filepath.Join(buildDir, "requirements.txt"), "--exclude", "setuptools", "pip", "wheel").Return(requirementsStale, nil)
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "uninstall", "-r", filepath.Join(depDir, "python", "requirements-stale.txt", "-y", "--exists-action=w"))
				Expect(supplier.UninstallUnusedDependencies()).To(Succeed())
				fileContents, err := ioutil.ReadFile(filepath.Join(depDir, "python", "requirements-stale.txt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(string(fileContents)).To(Equal(requirementsStale))
			})
		})

		Context("when requirements-declared.txt does not exist", func() {
			It("does nothing", func() {
				fileExists, err := libbuildpack.FileExists(filepath.Join(depDir, "python", "requirements-stale.txt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(fileExists).To(Equal(false))
				Expect(supplier.UninstallUnusedDependencies()).To(Succeed())
			})
		})
	})

	Describe("RunPipUnvendored", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
		})

		Context("requirements.txt exists in dep dir", func() {
			BeforeEach(func() {
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "python", "bin"), "bin")
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte{}, 0644)).To(Succeed())
			})

			It("Runs and outputs pip", func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--disable-pip-version-check")
				Expect(supplier.RunPipUnvendored()).To(Succeed())
			})
		})

		Context("requirements.txt exists in dep dir and pip install fails", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte{}, 0644)).To(Succeed())
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--disable-pip-version-check").Return(fmt.Errorf("exit 28"))
			})

			const proTip = "Running pip install failed. You need to include all dependencies in the vendor directory."

			It("does NOT alert the user", func() {
				Expect(supplier.RunPipUnvendored()).To(MatchError(fmt.Errorf("could not run pip: exit 28")))
				Expect(buffer.String()).ToNot(ContainSubstring(proTip))
			})
		})

		Context("requirements.txt is NOT in dep dir", func() {
			It("exits early", func() {
				Expect(supplier.RunPipUnvendored()).To(Succeed())
			})
		})

		Context("have index_url, find_links, allow_hosts exists in pydistutils.cfg file", func() {
			requirements :=
				`--index-url https://index-url
--extra-index-url https://extra-index-url1
--extra-index-url https://extra-index-url2
--trusted-host extra-index-url1
--trusted-host extra-index-url2
Flask==0.10.1
Jinja2==2.7.2
MarkupSafe==0.21
`
			BeforeEach(func() {
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "python", "bin"), "bin")
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte(requirements), 0644)).To(Succeed())
			})

			It("check index_url, find_links, allow_hosts values", func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--disable-pip-version-check")
				Expect(supplier.RunPipUnvendored()).To(Succeed())
				filePath := filepath.Join(os.Getenv("HOME"), ".pydistutils.cfg")
				fileContents, err := ioutil.ReadFile(filePath)
				Expect(err).ShouldNot(HaveOccurred())
				configMap, err := ParsePydistutils(string(fileContents))
				Expect(err).ToNot(HaveOccurred())

				Expect(configMap).Should(HaveKeyWithValue("index_url", []string{"https://index-url"}))
				// find_links is array of string
				Expect(configMap).Should(HaveKeyWithValue("find_links", []string{"https://extra-index-url1", "https://extra-index-url2"}))
				// allow_hosts is comma separated string
				Expect(configMap).Should(HaveKeyWithValue("allow_hosts", []string{"extra-index-url1,extra-index-url2"}))
			})
		})
	})

	Describe("RunPipVendored", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(depDir, 0755)).To(Succeed())
		})

		Context("requirements.txt exists in dep dir", func() {
			BeforeEach(func() {
				mockStager.EXPECT().LinkDirectoryInDepDir(filepath.Join(depDir, "python", "bin"), "bin")
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte{}, 0644)).To(Succeed())
				Expect(os.Mkdir(filepath.Join(buildDir, "vendor"), 0755)).To(Succeed())
			})

			It("installs the vendor directory", func() {
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "--no-build-isolation", "-h").Return(nil)
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--no-index", fmt.Sprintf("--find-links=file://%s/vendor", buildDir), "--disable-pip-version-check", "--no-build-isolation")
				Expect(supplier.RunPipVendored()).To(Succeed())
			})
		})

		Context("requirements.txt exists in dep dir and pip install fails", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "requirements.txt"), []byte{}, 0644)).To(Succeed())
				Expect(os.Mkdir(filepath.Join(buildDir, "vendor"), 0755)).To(Succeed())
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "--no-build-isolation", "-h").Return(nil)
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--no-index", fmt.Sprintf("--find-links=file://%s/vendor", buildDir), "--disable-pip-version-check", "--no-build-isolation").Return(fmt.Errorf("exit 28"))
				mockCommand.EXPECT().Execute(buildDir, gomock.Any(), gomock.Any(), "python", "-m", "pip", "install", "-r", filepath.Join(buildDir, "requirements.txt"), "--ignore-installed", "--exists-action=w", fmt.Sprintf("--src=%s/src", depDir), "--disable-pip-version-check").Return(fmt.Errorf("exit 28"))
			})

			const proTip = "Running pip install failed. You need to include all dependencies in the vendor directory."

			It("alerts the user", func() {
				Expect(supplier.RunPipVendored()).To(MatchError(fmt.Errorf("could not run pip: exit 28")))
				Expect(buffer.String()).To(ContainSubstring(proTip))
			})
		})

		Context("requirements.txt is NOT in dep dir", func() {
			It("exits early", func() {
				Expect(supplier.RunPipVendored()).To(Succeed())
			})
		})
	})

	Describe("CreateDefaultEnv", func() {
		It("writes an env file for PYTHONPATH", func() {
			mockStager.EXPECT().WriteEnvFile("PYTHONPATH", depDir)
			mockStager.EXPECT().WriteEnvFile("LIBRARY_PATH", filepath.Join(depDir, "lib"))
			mockStager.EXPECT().WriteEnvFile("PYTHONHASHSEED", "random")
			mockStager.EXPECT().WriteEnvFile("PYTHONUNBUFFERED", "1")
			mockStager.EXPECT().WriteEnvFile("LANG", "en_US.UTF-8")
			mockStager.EXPECT().WriteEnvFile("PYTHONHOME", filepath.Join(depDir, "python"))
			mockStager.EXPECT().WriteProfileD(gomock.Any(), gomock.Any())
			Expect(supplier.CreateDefaultEnv()).To(Succeed())
		})

		It("writes the profile.d", func() {
			mockStager.EXPECT().WriteEnvFile(gomock.Any(), gomock.Any()).AnyTimes()
			mockStager.EXPECT().WriteProfileD("python.sh", fmt.Sprintf(`export LANG=${LANG:-en_US.UTF-8}
export PYTHONHASHSEED=${PYTHONHASHSEED:-random}
export PYTHONPATH=$DEPS_DIR/%s
export PYTHONHOME=$DEPS_DIR/%s/python
export PYTHONUNBUFFERED=1
export FORWARDED_ALLOW_IPS='*'
export GUNICORN_CMD_ARGS=${GUNICORN_CMD_ARGS:-'--access-logfile -'}
`, depsIdx, depsIdx))
			Expect(supplier.CreateDefaultEnv()).To(Succeed())
		})

		Context("HasNltkData=true", func() {
			BeforeEach(func() {
				supplier.HasNltkData = true
			})
			It("writes an env file for NLTK_DATA", func() {
				mockStager.EXPECT().WriteEnvFile("NLTK_DATA", filepath.Join(depDir, "python", "nltk_data"))
				mockStager.EXPECT().WriteEnvFile(gomock.Any(), gomock.Any()).AnyTimes()

				mockStager.EXPECT().WriteProfileD(gomock.Any(), gomock.Any())

				Expect(supplier.CreateDefaultEnv()).To(Succeed())
			})

			It("writes the profile.d", func() {
				mockStager.EXPECT().WriteEnvFile(gomock.Any(), gomock.Any()).AnyTimes()
				mockStager.EXPECT().WriteProfileD("python.sh", gomock.Any()).Do(func(_, actual string) {
					expected := fmt.Sprintf("export NLTK_DATA=$DEPS_DIR/%s/python/nltk_data", depsIdx)
					Expect(actual).To(ContainSubstring(expected))
				})
				Expect(supplier.CreateDefaultEnv()).To(Succeed())
			})
		})
	})

	Describe("DownloadNLTKCorpora", func() {
		Context("NLTK not installed", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), "python", "-m", "nltk.downloader", "-h").Return(errors.New(""))
			})
			It("should not do anything", func() {
				Expect(supplier.DownloadNLTKCorpora()).To(Succeed())
				Expect(buffer.String()).To(Equal(""))
			})
		})

		Context("NLTK installed", func() {
			BeforeEach(func() {
				mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), "python", "-m", "nltk.downloader", "-h").Return(nil)
			})
			It("logs downloading", func() {
				Expect(supplier.DownloadNLTKCorpora()).To(Succeed())
				Expect(buffer.String()).To(ContainSubstring("Downloading NLTK corpora"))
				Expect(supplier.HasNltkData).To(BeFalse())
			})

			Context("nltk.txt is not in app", func() {
				BeforeEach(func() {
					Expect(filepath.Join(buildDir, "nltk.txt")).ToNot(BeARegularFile())
				})
				It("warns the user", func() {
					Expect(supplier.DownloadNLTKCorpora()).To(Succeed())
					Expect(buffer.String()).To(ContainSubstring("nltk.txt not found, not downloading any corpora"))
					Expect(supplier.HasNltkData).To(BeFalse())
				})
			})

			Context("nltk.txt exists in app", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(buildDir, "nltk.txt"), []byte("brown\nred\n"), 0644)).To(Succeed())
				})
				It("downloads nltk", func() {
					mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), "python", "-m", "nltk.downloader", "-d", filepath.Join(depDir, "python", "nltk_data"), "brown", "red").Return(nil)

					Expect(supplier.DownloadNLTKCorpora()).To(Succeed())

					Expect(buffer.String()).To(ContainSubstring("Downloading NLTK packages: brown red"))
					Expect(supplier.HasNltkData).To(BeTrue())
				})
			})
		})
	})

	Describe("SetupCacheDir", func() {
		AfterEach(func() { os.Unsetenv("XDG_CACHE_HOME") })

		It("Sets pip's cache directory", func() {
			mockStager.EXPECT().WriteEnvFile("XDG_CACHE_HOME", filepath.Join(cacheDir, "pip_cache"))
			Expect(supplier.SetupCacheDir()).To(Succeed())
			Expect(os.Getenv("XDG_CACHE_HOME")).To(Equal(filepath.Join(cacheDir, "pip_cache")))
		})
	})
})
