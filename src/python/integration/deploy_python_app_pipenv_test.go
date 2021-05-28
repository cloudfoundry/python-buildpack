package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deploying a flask web app", func() {
	var app *cutlass.App

	BeforeEach(func() {
		if isSerialTest {
			Skip("Skipping parallel tests")
		}
	})

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("app has Pipfile.lock and no requirements.txt or runtime.txt", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("flask_python_3_pipenv"))
			app.SetEnv("BP_DEBUG", "1")
			PushAppAndConfirm(app)
		})

		It("deploys a pipenv app", func() {
			By("getting the python version from pipfile.lock")
			Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.6."))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))

			By("generating a requirements.txt without updating the pipfile.lock packages")
			Expect(app.Stdout.String()).To(ContainSubstring("Generating 'requirements.txt' from Pipfile.lock"))
			Expect(app.GetBody("/versions")).To(ContainSubstring("Gunicorn version: 19.3.0"))
			Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
		})
	})

	Context("buildpack is cached", func() {
		Context("python 3", func() {
			Context("app is completely vendored", func() {
				BeforeEach(func() {
					if !cutlass.Cached {
						Skip("Running cached tests")
					}
					app = cutlass.New(Fixtures("flask_python_3_pipenv_vendored"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("should work", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
				})

				AssertNoInternetTraffic("flask_python_3_pipenv_vendored")
			})
			Context("app is missing a dependency", func() {
				BeforeEach(func() {
					if !cutlass.Cached {
						Skip("Running cached tests")
					}
					app = cutlass.New(Fixtures("flask_python_3_pipenv_vendored_incomplete"))
					app.SetEnv("BP_DEBUG", "1")
				})

				It("should work by downloading the missing dependency", func() {
					PushAppAndConfirm(app)
					Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
				})
			})
		})
	})

	Context("no Pipfile", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("no_deps"))
			app.Buildpacks = []string{"python_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("deploys without downloading pipenv", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).NotTo(ContainSubstring("Installing pipenv"))
			Expect(app.GetBody("/gg")).To(ContainSubstring("Here is your output for /gg"))
		})
	})

	Context("When there is a requirements.txt and a Pipfile", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("pipfile_and_requirements"))
			app.Buildpacks = []string{"python_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("deploys without downloading pipenv", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).NotTo(ContainSubstring("Installing pipenv"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with no pipenv!"))
		})
	})
})
