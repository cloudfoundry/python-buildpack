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
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "miniconda_python_2"))
			app.Disk = "2G"
			app.Memory = "1G"
		})

		It("deploys", func() {
			PushAppAndConfirm(app)

			body, err := app.GetBody("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.10.4"))
			Expect(body).To(ContainSubstring("scipy: 0.17.0"))
			Expect(body).To(ContainSubstring("sklearn: 0.17.1"))
			Expect(body).To(ContainSubstring("pandas: 0.18.0"))
			Expect(body).To(ContainSubstring("python-version2"))
		})

		AssertUsesProxyDuringStagingIfPresent("miniconda_python_2")
	})

	Context("an app that uses miniconda and python 3", func() {
		var fixtureDir string
		BeforeEach(func() {
			var err error
			fixtureDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "miniconda_python_3"))
			Expect(err).ToNot(HaveOccurred())
			app = cutlass.New(fixtureDir)
			app.Disk = "2G"
			app.Memory = "1G"
		})
		AfterEach(func() { _ = os.RemoveAll(fixtureDir) })

		It("keeps track of environment.yml", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("scipy"))

			body, err := app.GetBody("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(body).To(ContainSubstring("numpy: 1.10.4"))
			Expect(body).To(ContainSubstring("scipy: 0.17.0"))
			Expect(body).To(ContainSubstring("sklearn: 0.17.1"))
			Expect(body).To(ContainSubstring("pandas: 0.18.0"))
			Expect(body).To(ContainSubstring("python-version3"))
		})

		It("doesn't re-download unchanged dependencies", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("scipy"))

			app.Stdout.Reset()

			PushAppAndConfirm(app)
			// Check that scipy was not re-installed in the logs
			Expect(app.Stdout.String()).ToNot(ContainSubstring("scipy"))
		})

		It("it updates dependencies if environment.yml changes", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("numpy: 1.10.4"))
			Expect(app.GetBody("/")).ToNot(ContainSubstring("numpy: 1.11.0"))

			input, err := ioutil.ReadFile(filepath.Join(fixtureDir, "environment.yml"))
			Expect(err).ToNot(HaveOccurred())
			output := strings.Replace(string(input), "numpy=1.10.4", "numpy=1.11.0", 1)
			Expect(ioutil.WriteFile(filepath.Join(fixtureDir, "environment.yml"), []byte(output), 0644)).To(Succeed())

			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("numpy: 1.11.0"))
		})

		AssertUsesProxyDuringStagingIfPresent("miniconda_python_3")
	})

	Context("an app that uses miniconda and specifies python 2 in runtime.txt but python3 in the environment.yml", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "miniconda_python_2_3"))
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
