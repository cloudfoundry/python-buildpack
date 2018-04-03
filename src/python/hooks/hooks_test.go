package hooks_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"python/hooks"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hooks", func() {
	var (
		err      error
		buildDir string
		stager   *libbuildpack.Stager
		hook     libbuildpack.Hook
		buffer   *bytes.Buffer
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger := libbuildpack.NewLogger(ansicleaner.New(buffer))

		args := []string{buildDir, "", "/tmp/not-exist", "9"}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		hook = &hooks.AppHook{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(buildDir)).To(Succeed())
	})

	Describe("BeforeCompile", func() {
		Context("bin/pre_compile exists", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(filepath.Join(buildDir, "bin"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "bin", "pre_compile"), []byte("#!/usr/bin/env bash\n\necho -n jane > fred.txt\n"), 0644)).To(Succeed())
			})

			It("changes file to executable", func() {
				Expect(hook.BeforeCompile(stager)).To(Succeed())
				fileInfo, err := os.Stat(filepath.Join(buildDir, "bin", "pre_compile"))
				Expect(err).ToNot(HaveOccurred())
				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0755)))
				Expect(buffer.String()).To(ContainSubstring("Running pre_compile hook"))
			})

			It("runs successfully", func() {
				Expect(filepath.Join(buildDir, "fred.txt")).ToNot(BeARegularFile())

				Expect(hook.BeforeCompile(stager)).To(Succeed())

				Expect(filepath.Join(buildDir, "fred.txt")).To(BeARegularFile())
				Expect(ioutil.ReadFile(filepath.Join(buildDir, "fred.txt"))).To(Equal([]byte("jane")))
			})
		})

		It("runs successfully when bin/pre_compile exists but has no shebang", func() {
			Expect(os.Mkdir(filepath.Join(buildDir, "bin"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildDir, "bin", "pre_compile"), []byte("echo 'Hello world!'"), 0644)).To(Succeed())
			Expect(hook.BeforeCompile(stager)).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("Hello world!"))
		})

		It("does nothing when bin/pre_compile does NOT exist", func() {
			Expect(hook.BeforeCompile(stager)).To(Succeed())
			Expect(buffer.String()).NotTo(ContainSubstring("Running pre_compile hook"))
		})
	})

	Describe("AfterCompile", func() {
		Context("bin/post_compile exists", func() {
			BeforeEach(func() {
				Expect(os.Mkdir(filepath.Join(buildDir, "bin"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(buildDir, "bin", "post_compile"), []byte("#!/usr/bin/env bash\n\necho -n john > fred.txt\n"), 0644)).To(Succeed())
			})
			It("changes file to executable", func() {
				Expect(hook.AfterCompile(stager)).To(Succeed())
				fileInfo, err := os.Stat(filepath.Join(buildDir, "bin", "post_compile"))
				Expect(err).ToNot(HaveOccurred())
				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0755)))
				Expect(buffer.String()).To(ContainSubstring("Running post_compile hook"))
			})

			It("runs successfully", func() {
				Expect(filepath.Join(buildDir, "fred.txt")).ToNot(BeARegularFile())

				Expect(hook.AfterCompile(stager)).To(Succeed())

				Expect(filepath.Join(buildDir, "fred.txt")).To(BeARegularFile())
				Expect(ioutil.ReadFile(filepath.Join(buildDir, "fred.txt"))).To(Equal([]byte("john")))
			})
		})

		It("runs successfully when bin/post_compile exists but has no shebang", func() {
			Expect(os.Mkdir(filepath.Join(buildDir, "bin"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(buildDir, "bin", "post_compile"), []byte("echo 'Hello world!'"), 0644)).To(Succeed())
			Expect(hook.AfterCompile(stager)).To(Succeed())
			Expect(buffer.String()).To(ContainSubstring("Hello world!"))
		})

		It("does nothing when bin/post_compile does NOT exist", func() {
			Expect(hook.AfterCompile(stager)).To(Succeed())
			Expect(buffer.String()).NotTo(ContainSubstring("Running post_compile hook"))
		})
	})
})
