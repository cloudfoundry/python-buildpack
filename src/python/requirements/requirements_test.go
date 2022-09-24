package requirements

import (
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
		tempDir, err = os.MkdirTemp("", "requirements")
		Expect(err).NotTo(HaveOccurred())
		req = Reqs{}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tempDir)).To(Succeed())
	})

	Describe("FindAnyPackage", func() {
		Context("succeed", func() {

			Context("single requirements.txt file", func() {
				BeforeEach(func() {
					Expect(os.WriteFile(filepath.Join(tempDir, "requirements.txt"), []byte(`package0
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

			Context("multiple requirements.txt files", func() {
				Context("packages are in a recursive requirements.txt file", func() {
					BeforeEach(func() {
						err := os.Mkdir(filepath.Join(tempDir, "other_folder"), 0755)
						Expect(err).NotTo(HaveOccurred())

						Expect(os.WriteFile(filepath.Join(tempDir, "requirements.txt"), []byte(`package0
package3!=3.0.0
package4~=4.0.0
-r requirements1.txt`), 0644)).To(Succeed())

						Expect(os.WriteFile(filepath.Join(tempDir, "requirements1.txt"), []byte(`package6[test] == 6.0.0
-r other_folder/requirements2.txt`), 0644)).To(Succeed())
						Expect(os.WriteFile(filepath.Join(tempDir, "other_folder", "requirements2.txt"), []byte(`package2>=2.0.0`), 0644)).To(Succeed())
					})

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
		})
	})

	Describe("FindStalePackages", func() {
		Context("succeed", func() {
			BeforeEach(func() {
				Expect(os.WriteFile(filepath.Join(tempDir, "req-old.txt"), []byte(`package0
package1==2.0.0
package2
package3==3.0.0
package4
package5!=4.0.0
`), 0644)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(tempDir, "req-new.txt"), []byte(`package0
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
	})
})
