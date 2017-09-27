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

	Context("app has Pipfile.lock and no requirements.txt or runtime.txt", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3_pipenv"))
			PushAppAndConfirm(app)
		})

		It("gets the python version from pipfile.lock and generates a runtime.txt", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.6."))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
		})

		It("generates a requirements.txt", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Generating 'requirements.txt' with pipenv"))
		})
	})

	Context("buildpack is cached", func() {
		BeforeEach(func() {
			if !cutlass.Cached {
				Skip("Running cached tests")
			}
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3_pipenv_vendored"))
		})

		It("should work", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
		})

		AssertNoInternetTraffic("flask_python_3_pipenv_vendored")
	})
})
