package pyfinder_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	. "python/pyfinder"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ManagePyFinder", func() {
	var (
		tempDir string
		finder  ManagePyFinder
		err     error
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "pyfinder")
		Expect(err).NotTo(HaveOccurred())
		finder = ManagePyFinder{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	Describe("FindManagePy", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(tempDir, "a", "b"), 0755)).To(Succeed())
		})

		Context("no manage.py exists", func() {
			It("returns an error", func() {
				path, err := finder.FindManagePy(tempDir)
				Expect(err).To(HaveOccurred())
				Expect(path).To(Equal(""))
			})
		})

		Context("manage.py exists 4 directories down", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(tempDir, "a", "b", "toodeep"), 0755)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tempDir, "a", "b", "toodeep", "manage.py"), []byte("hello"), 0644)).To(Succeed())
			})

			It("returns an error", func() {
				path, err := finder.FindManagePy(tempDir)
				Expect(err).To(HaveOccurred())
				Expect(path).To(Equal(""))
			})
		})

		Context("when a manage.py exists 3 directories down", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tempDir, "a", "b", "manage.py"), []byte("hello"), 0644)).To(Succeed())
			})

			It("finds the manage.py 3 directories deep", func() {
				path, err := finder.FindManagePy(tempDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(tempDir, "a", "b", "manage.py")))
			})

			Context("and another exists three directories down", func() {
				var files []string
				BeforeEach(func() {
					files = []string{filepath.Join(tempDir, "a", "b", "manage.py"), filepath.Join(tempDir, "a", "c", "manage.py")}
					Expect(os.MkdirAll(filepath.Join(tempDir, "a", "c"), 0755)).To(Succeed())
					Expect(ioutil.WriteFile(filepath.Join(tempDir, "a", "c", "manage.py"), []byte("hello"), 0644)).To(Succeed())
				})
				It("returns either", func() {
					path, err := finder.FindManagePy(tempDir)
					Expect(err).NotTo(HaveOccurred())
					Expect(files).To(ContainElement(path))
				})
			})

			Context("and another exists 2 directories down", func() {
				BeforeEach(func() {
					Expect(ioutil.WriteFile(filepath.Join(tempDir, "a", "manage.py"), []byte("hello"), 0644)).To(Succeed())
				})
				It("finds the manage.py 2 directories deep", func() {
					path, err := finder.FindManagePy(tempDir)
					Expect(err).NotTo(HaveOccurred())
					Expect(path).To(Equal(filepath.Join(tempDir, "a", "manage.py")))
				})

				Context("and another exists 1 directory down", func() {
					BeforeEach(func() {
						Expect(ioutil.WriteFile(filepath.Join(tempDir, "manage.py"), []byte("hello"), 0644)).To(Succeed())
					})
					It("finds the manage.py 1 directories deep", func() {
						path, err := finder.FindManagePy(tempDir)
						Expect(err).NotTo(HaveOccurred())
						Expect(path).To(Equal(filepath.Join(tempDir, "manage.py")))
					})
				})
			})
		})
	})
})
