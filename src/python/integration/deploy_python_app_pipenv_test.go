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
			app.SetEnv("BP_DEBUG", "1")
			PushAppAndConfirm(app)
		})

		It("gets the python version from pipfile.lock and generates a runtime.txt", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing python 3.6."))
			Expect(app.Stdout.String()).To(ContainSubstring("Installing pipenv"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with pipenv!"))
			Expect(app.Stdout.String()).To(ContainSubstring("Dir checksum unchanged"))
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

	Context("no Pipfile", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "no_deps"))
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
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "pipfile_and_requirements"))
			app.Buildpacks = []string{"python_buildpack"}
			app.SetEnv("BP_DEBUG", "1")
		})

		It("deploys without downloading pipenv", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).NotTo(ContainSubstring("Installing pipenv"))
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World with no pipenv!"))
		})
	})

	Context("When there python 3.3.* is used", func() {
		BeforeEach(func() {
			app = cutlass.New(filepath.Join(bpDir, "fixtures", "flask_python_3_3_pipenv"))
			app.Buildpacks = []string{"python_buildpack"}
		})

		It("returns an error", func() {
			Expect(app.Push()).ToNot(Succeed())
			Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

			Expect(app.Stdout.String()).To(ContainSubstring("Could not install pipenv: pipenv does not support python 3.3.x"))
		})
	})
})
