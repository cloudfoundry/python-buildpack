package integration_test

import (
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploying a flask web app", func() {
	var app *cutlass.App

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	It("app has Pipfile.lock and no requirements.txt or runtime.txt", func() {
		app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_python_3_pipenv"))
		PushAppAndConfirm(app)

		By("it gets the python version from pipfile.lock", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing python-3.6."))
		})

		By("uses pipenv to generate a requirements.txt", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Generating 'requirements.txt' with pipenv"))
		})

		Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
	})

	Context("buildpack is cached", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("but running uncached tests")
			}
		})

		It("deploys without hitting the internet", func() {
			app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_python_3_pipenv_vendored"))
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("Generating 'requirements.txt' with pipenv"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
		})

		AssertNoInternetTraffic("flask_python_3_pipenv_vendored")
	})
})
