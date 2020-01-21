package integration_test

import (
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

	BeforeEach(func() {
		if !isMinicondaTest {
			Skip("Skipping miniconda tests")
		}
	})

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	CleanAnsi := func(s string) string {
		r := strings.NewReplacer("\033[31;1m", "", "\033[33;1m", "", "\033[34;1m", "", "\033[0m", "")
		return r.Replace(s)
	}

	Context("an app that uses miniconda and python 2", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("miniconda_python_2"))
			app.Disk = "2G"
			app.Memory = "1G"
		})

		It("deploys", func() {
			PushAppAndConfirm(app)

			body, err := app.GetBody("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.16.5"))
		})

		AssertUsesProxyDuringStagingIfPresent("miniconda_python_2")
	})

	Context("an app that uses miniconda and python 3", func() {
		var fixtureDir string
		BeforeEach(func() {
			var err error
			fixtureDir, err = cutlass.CopyFixture(Fixtures("miniconda_python_3"))
			Expect(err).ToNot(HaveOccurred())
			app = cutlass.New(fixtureDir)
			app.Disk = "2G"
			app.Memory = "1G"
			app.Buildpacks = []string{"python_buildpack"}
		})
		AfterEach(func() { _ = os.RemoveAll(fixtureDir) })

		It("keeps track of environment.yml", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("numpy"))

			body, err := app.GetBody("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.15.2"))
			Expect(body).To(ContainSubstring("python-version3"))
		})

		It("doesn't re-download unchanged dependencies", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("numpy"))

			app.Stdout.Reset()

			PushAppAndConfirm(app)
			// Check that numpy was not re-installed in the logs
			Expect(app.Stdout.String()).ToNot(ContainSubstring("numpy"))
		})

		It("it updates dependencies if environment.yml changes", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("numpy: 1.15.2"))
			Expect(app.GetBody("/")).ToNot(ContainSubstring("numpy: 1.15.0"))

			input, err := ioutil.ReadFile(filepath.Join(fixtureDir, "environment.yml"))
			Expect(err).ToNot(HaveOccurred())
			output := strings.Replace(string(input), "numpy=1.15.2", "numpy=1.15.0", 1)
			Expect(ioutil.WriteFile(filepath.Join(fixtureDir, "environment.yml"), []byte(output), 0644)).To(Succeed())

			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("numpy: 1.15.0"))
		})

		AssertUsesProxyDuringStagingIfPresent("miniconda_python_3")
	})

	Context("an app that uses miniconda and specifies python 2 in runtime.txt but python3 in the environment.yml", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("miniconda_python_2_3"))
			app.Disk = "2G"
			app.Memory = "1G"
		})

		It("deploys", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("python-version3"))
			Expect(CleanAnsi(app.Stdout.String())).To(ContainSubstring("**WARNING** you have specified the version of Python runtime both in 'runtime.txt' and 'environment.yml'. You should remove one of the two versions"))
		})

		AssertUsesProxyDuringStagingIfPresent("miniconda_python_2_3")
	})
})
