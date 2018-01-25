package conda_test

import (
	"bytes"
	io "io"
	"io/ioutil"
	"os"
	"path/filepath"

	"python/conda"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=conda.go --destination=mocks_test.go --package=conda_test

var _ = Describe("Conda", func() {
	var (
		err          error
		buildDir     string
		cacheDir     string
		depsDir      string
		depsIdx      string
		depDir       string
		subject      *conda.Conda
		logger       *libbuildpack.Logger
		buffer       *bytes.Buffer
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
		mockStager   *MockStager
		mockCommand  *MockCommand
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
		mockStager = NewMockStager(mockCtrl)
		mockStager.EXPECT().BuildDir().AnyTimes().Return(buildDir)
		mockStager.EXPECT().CacheDir().AnyTimes().Return(cacheDir)
		mockStager.EXPECT().DepDir().AnyTimes().Return(depDir)
		mockStager.EXPECT().DepsIdx().AnyTimes().Return(depsIdx)
		mockCommand = NewMockCommand(mockCtrl)

		buffer = new(bytes.Buffer)
		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		subject = conda.New(mockManifest, mockStager, mockCommand, logger)
	})

	AfterEach(func() {
		mockCtrl.Finish()
		Expect(os.RemoveAll(buildDir)).To(Succeed())
		Expect(os.RemoveAll(cacheDir)).To(Succeed())
		Expect(os.RemoveAll(depsDir)).To(Succeed())
	})

	Describe("Version", func() {
		Context("runtime.txt specifies python 2", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "runtime.txt"), []byte("python-2.6.3"), 0644)).To(Succeed())
			})
			It("returns 'miniconda2'", func() {
				Expect(subject.Version()).To(Equal("miniconda2"))
			})
		})
		Context("runtime.txt specifies python 3", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "runtime.txt"), []byte("python-3.2.3"), 0644)).To(Succeed())
			})
			It("returns 'miniconda3'", func() {
				Expect(subject.Version()).To(Equal("miniconda3"))
			})
		})
		Context("runtime.txt does not exist", func() {
			It("returns 'miniconda2'", func() {
				Expect(subject.Version()).To(Equal("miniconda2"))
			})
		})
	})

	Describe("Install", func() {
		It("downloads and installs miniconda version", func() {
			mockManifest.EXPECT().InstallOnlyVersion("Miniconda7", gomock.Any()).Do(func(_, path string) {
				Expect(ioutil.WriteFile(path, []byte{}, 0644)).To(Succeed())
			})
			mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

			Expect(subject.Install("Miniconda7")).To(Succeed())
		})

		It("make downloaded file executable", func() {
			mockManifest.EXPECT().InstallOnlyVersion("Miniconda7", gomock.Any()).Do(func(_, path string) {
				Expect(ioutil.WriteFile(path, []byte{}, 0644)).To(Succeed())
			})
			mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), gomock.Any(), "-b", "-p", filepath.Join(depDir, "conda")).Do(func(_ string, _, _ io.Writer, path, _, _, _ string) {
				fi, err := os.Lstat(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(fi.Mode()).To(Equal(os.FileMode(0755)))
			})

			Expect(subject.Install("Miniconda7")).To(Succeed())
		})

		It("deletes installer", func() {
			var installerPath string
			mockManifest.EXPECT().InstallOnlyVersion("Miniconda7", gomock.Any()).Do(func(_, path string) {
				Expect(ioutil.WriteFile(path, []byte{}, 0644)).To(Succeed())
				installerPath = path
			})
			mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), gomock.Any(), "-b", "-p", filepath.Join(depDir, "conda")).Do(func(_ string, _, _ io.Writer, path, _, _, _ string) {
				Expect(path).To(Equal(installerPath))
			})

			Expect(subject.Install("Miniconda7")).To(Succeed())

			Expect(installerPath).ToNot(BeARegularFile())
		})

	})

	Describe("UpdateAndClean", func() {
		AfterEach(func() {
			os.Unsetenv("BP_DEBUG")
		})
		Context("BP_DEBUG == false", func() {
			It("calls update and clean on conda (with quiet flag)", func() {
				mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), filepath.Join(depDir, "conda", "bin", "conda"), "env", "update", "--quiet", "-n", "dep_env", "-f", filepath.Join(buildDir, "environment.yml"))
				mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), filepath.Join(depDir, "conda", "bin", "conda"), "clean", "-pt")
				Expect(subject.UpdateAndClean()).To(Succeed())
			})
		})
		Context("BP_DEBUG == true", func() {
			BeforeEach(func() {
				os.Setenv("BP_DEBUG", "1")
			})
			It("calls update and clean on conda (with debug and verbose flags)", func() {
				mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), filepath.Join(depDir, "conda", "bin", "conda"), "env", "update", "--debug", "--verbose", "-n", "dep_env", "-f", filepath.Join(buildDir, "environment.yml"))
				mockCommand.EXPECT().Execute("/", gomock.Any(), gomock.Any(), filepath.Join(depDir, "conda", "bin", "conda"), "clean", "-pt")
				Expect(subject.UpdateAndClean()).To(Succeed())
			})
		})
	})

	It("ProfileD", func() {
		Expect(subject.ProfileD()).To(Equal(`grep -rlI ` + depDir + ` $DEPS_DIR/13/conda | xargs sed -i -e "s|` + depDir + `|$DEPS_DIR/13|g"
source activate dep_env
`))
	})

	Describe("SaveCache", func() {
		It("copies the conda envs dir to cache", func() {
			mockCommand.EXPECT().Output("/", "cp", "-Rl", filepath.Join(depDir, "conda", "envs"), filepath.Join(cacheDir, "envs"))

			Expect(subject.SaveCache()).To(Succeed())
		})
		It("stores dep dir in cache as conda_prefix", func() {
			mockCommand.EXPECT().Output(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			Expect(subject.SaveCache()).To(Succeed())

			Expect(filepath.Join(cacheDir, "conda_prefix")).To(BeARegularFile())
			Expect(ioutil.ReadFile(filepath.Join(cacheDir, "conda_prefix"))).To(Equal([]byte(depDir)))
		})
	})

	Describe("RestoreCache", func() {
		Context("no cache", func() {
			It("does nothing", func() {
				Expect(subject.RestoreCache()).To(Succeed())
			})
		})
		Context("envs cache exists", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "conda_prefix"), []byte("/old/dep/dir\n"), 0644)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(cacheDir, "envs", "dir1", "dir2"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "envs", "dir1", "dir2", "file"), []byte("contents"), 0644)).To(Succeed())

				Expect(os.MkdirAll(filepath.Join(depDir, "conda", "envs", "existing"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(depDir, "conda", "envs", "existing", "file"), []byte("contents"), 0644)).To(Succeed())
			})
			It("moves copies cache envs directories to conda directory", func() {
				Expect(subject.RestoreCache()).To(Succeed())

				Expect(filepath.Join(depDir, "conda", "envs", "dir1", "dir2", "file")).To(BeARegularFile())
				Expect(ioutil.ReadFile(filepath.Join(depDir, "conda", "envs", "dir1", "dir2", "file"))).To(Equal([]byte("contents")))
			})
			It("does not alter existing files in conda envs", func() {
				Expect(subject.RestoreCache()).To(Succeed())

				Expect(filepath.Join(depDir, "conda", "envs", "existing", "file")).To(BeARegularFile())
				Expect(ioutil.ReadFile(filepath.Join(depDir, "conda", "envs", "existing", "file"))).To(Equal([]byte("contents")))
			})
			It("converts old depDir to new depDir", func() {
				Expect(ioutil.WriteFile(filepath.Join(cacheDir, "envs", "dir1", "dir2", "file"), []byte("run /old/dep/dir/conda"), 0644)).To(Succeed())

				Expect(subject.RestoreCache()).To(Succeed())

				Expect(filepath.Join(depDir, "conda", "envs", "dir1", "dir2", "file")).To(BeARegularFile())
				Expect(ioutil.ReadFile(filepath.Join(depDir, "conda", "envs", "dir1", "dir2", "file"))).To(Equal([]byte("run " + depDir + "/conda")))
			})
		})
	})
})
