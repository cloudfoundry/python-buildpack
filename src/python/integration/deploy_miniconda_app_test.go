package integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CF Python Buildpack", func() {
	var app *cutlass.App
	var fixtureDir string

	BeforeEach(func() {
		if !isSerialTest {
			Skip("Skipping serial tests")
		}

		var err error
		fixtureDir, err = cutlass.CopyFixture(Fixtures("miniconda_python_3"))
		Expect(err).ToNot(HaveOccurred())
		app = cutlass.New(fixtureDir)
		app.Buildpacks = []string{"python_buildpack"}
		app.Disk = "2G"
		app.Memory = "1G"
	})

	AfterEach(func() {
		Expect(os.RemoveAll(fixtureDir)).To(Succeed())

		if app != nil {
			Expect(app.Destroy()).To(Succeed())
			app = nil
		}
	})

	Context("an app that uses miniconda and python 3", func() {
		It("keeps track of environment.yml", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("numpy"))

			body, err := app.GetBody("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.21.2"))
			Expect(body).To(ContainSubstring("python-version3"))
		})

		It("it updates dependencies if environment.yml changes", func() {
			fmt.Fprintln(GinkgoWriter, "Pushing original app...")
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("numpy: 1.21.2"))
			Expect(app.GetBody("/")).ToNot(ContainSubstring("numpy: 1.20.2"))

			input, err := ioutil.ReadFile(filepath.Join(fixtureDir, "environment.yml"))
			Expect(err).ToNot(HaveOccurred())
			output := strings.Replace(string(input), "numpy=1.21.2", "numpy=1.20.2", 1)
			Expect(ioutil.WriteFile(filepath.Join(fixtureDir, "environment.yml"), []byte(output), 0644)).To(Succeed())

			fmt.Fprintln(GinkgoWriter, "Pushing updated app...")
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("numpy: 1.20.2"))
		})

		AssertUsesProxyDuringStagingIfPresent("miniconda_python_3")
	})

	Context("an app that has no runtime.txt", func() {
		BeforeEach(func() {
			Expect(os.RemoveAll(filepath.Join(fixtureDir, "runtime.txt"))).To(Succeed())
		})

		It("works as expected", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("numpy"))

			body, err := app.GetBody("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.21.2"))
			Expect(body).To(ContainSubstring("python-version3"))
		})
	})
})
