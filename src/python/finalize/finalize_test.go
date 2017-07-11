package finalize_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"python/finalize"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	// . "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=finalize.go --destination=mocks_test.go --package=finalize_test

var _ = Describe("Finalize", func() {
	var (
		err          error
		buildDir     string
		depsDir      string
		depsIdx      string
		binDir       string
		finalizer    *finalize.Finalizer
		logger       *libbuildpack.Logger
		buffer       *bytes.Buffer
		mockCtrl     *gomock.Controller
		mockCommand  *MockCommand
		mockManifest *MockManifest
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "python-buildpack.deps.")
		Expect(err).To(BeNil())

		depsIdx = "7"
		Expect(os.MkdirAll(filepath.Join(depsDir, depsIdx), 0755)).To(Succeed())

		binDir = filepath.Join(depsDir, depsIdx, "bin")

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger(ansicleaner.New(buffer))

		mockCtrl = gomock.NewController(GinkgoT())
		mockCommand = NewMockCommand(mockCtrl)
		mockManifest = NewMockManifest(mockCtrl)

		args := []string{buildDir, "", depsDir, depsIdx}
		stager := libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		finalizer = &finalize.Finalizer{
			Stager:   stager,
			Manifest: mockManifest,
			Command:  mockCommand,
			Log:      logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depsDir)
		Expect(err).To(BeNil())
	})

	Describe("RewriteShebang", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(binDir, 0755)).To(Succeed())
		})

		Context("single file with specific path shebang", func() {
			BeforeEach(func() {
				contents := fmt.Sprintf("#!%s/python\n\nPython code\n", binDir)
				Expect(ioutil.WriteFile(filepath.Join(binDir, "afile"), []byte(contents), 0755)).To(Succeed())
			})

			It("converted to #!/usr/bin/env python", func() {
				Expect(finalizer.RewriteShebang()).To(Succeed())
				contents, err := ioutil.ReadFile(filepath.Join(binDir, "afile"))
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal("#!/usr/bin/env python\n\nPython code\n"))
			})
		})

		Context("symlink to file with specific path shebang", func() {
			var otherdir string
			BeforeEach(func() {
				otherdir = filepath.Join(depsDir, depsIdx, "adir")
				Expect(os.MkdirAll(otherdir, 0755)).To(Succeed())
				contents := fmt.Sprintf("#!%s/python\n\nPython code\n", binDir)
				Expect(ioutil.WriteFile(filepath.Join(otherdir, "afile"), []byte(contents), 0715)).To(Succeed())
				Expect(os.Symlink(filepath.Join(otherdir, "afile"), filepath.Join(binDir, "afile"))).To(Succeed())
			})

			It("converted to #!/usr/bin/env python", func() {
				Expect(finalizer.RewriteShebang()).To(Succeed())

				contents, err := ioutil.ReadFile(filepath.Join(otherdir, "afile"))
				Expect(err).To(BeNil())
				Expect(string(contents)).To(Equal("#!/usr/bin/env python\n\nPython code\n"))
			})

			It("retains permissions", func() {
				Expect(finalizer.RewriteShebang()).To(Succeed())

				info, err := os.Stat(filepath.Join(otherdir, "afile"))
				Expect(err).To(BeNil())
				Expect(info.Mode()).To(Equal(os.FileMode(0715)))
			})
		})
	})
})
