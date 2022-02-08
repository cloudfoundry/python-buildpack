package requirements

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reqs", func() {
	var (
		tempDir string
		req     Reqs
		err     error
	)

	BeforeEach(func() {
		tempDir, err = ioutil.TempDir("", "requirements")
		Expect(err).NotTo(HaveOccurred())
		req = Reqs{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	Describe("FindAnyPackage", func() {
		Context("succeed", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tempDir, "requirements.txt"), []byte(`package0
package2>=2.0.0
package3!=3.0.0
package4~=4.0.0
package6[test]==6.0.0
`), 0644)).To(Succeed())
			})

			Context("package is not in requirements.txt file", func() {
				It("returns false and nil error", func() {
					exists, err := req.FindAnyPackage(tempDir, "package1")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())

					exists, err = req.FindAnyPackage(tempDir, "package5")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeFalse())
				})
			})

			Context("package is in requirements.txt file", func() {
				It("returns true and nil error", func() {
					exists, err := req.FindAnyPackage(tempDir, "package2")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())

					exists, err = req.FindAnyPackage(tempDir, "package6[test]")
					Expect(err).ToNot(HaveOccurred())
					Expect(exists).To(BeTrue())
				})
			})
		})

		Context("failure", func() {
			Context("error opening requirements.txt file", func() {
				It("returns the error", func() {
					_, err := req.FindAnyPackage("invalid-directory", "package0")
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})

	Describe("FindStalePackages", func() {
		Context("succeed", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tempDir, "req-old.txt"), []byte(`package0
package1==2.0.0
package2
package3==3.0.0
package4
package5!=4.0.0
`), 0644)).To(Succeed())
				Expect(ioutil.WriteFile(filepath.Join(tempDir, "req-new.txt"), []byte(`package0
package1==2.0.0
`), 0644)).To(Succeed())
			})

			It("returns stale packages", func() {
				stale, err := req.FindStalePackages(filepath.Join(tempDir, "req-old.txt"), filepath.Join(tempDir, "req-new.txt"))
				Expect(err).ToNot(HaveOccurred())
				Expect(stale).To(ConsistOf("package2", "package3==3.0.0", "package4", "package5!=4.0.0"))
			})

			It("returns stale packages that are not in the excluded list", func() {
				stale, err := req.FindStalePackages(filepath.Join(tempDir, "req-old.txt"), filepath.Join(tempDir, "req-new.txt"),
					"package2", "package3")
				Expect(err).ToNot(HaveOccurred())
				Expect(stale).To(ConsistOf("package4", "package5!=4.0.0"))
			})
		})

		Context("failure", func() {
			BeforeEach(func() {
				Expect(ioutil.WriteFile(filepath.Join(tempDir, "valid-req.txt"), []byte(`package0`), 0644)).To(Succeed())
			})

			Context("error opening old requirements file", func() {
				It("returns the error", func() {
					_, err := req.FindStalePackages(filepath.Join(tempDir, "missing.txt"), filepath.Join(tempDir, "valid-req.txt"))
					Expect(err).To(HaveOccurred())
				})
			})

			Context("error opening new requirements file", func() {
				It("returns the error", func() {
					_, err := req.FindStalePackages(filepath.Join(tempDir, "valid-req.txt"), filepath.Join(tempDir, "missing.txt"))
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
