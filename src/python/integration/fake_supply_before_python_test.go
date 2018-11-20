package integration_test

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("running supply buildpacks before the python buildpack", func() {
	var app *cutlass.App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("a simple app is pushed once", func() {
		BeforeEach(func() {
			if ok, err := cutlass.ApiGreaterThan("2.65.1"); err != nil || !ok {
				Skip("API version does not have multi-buildpack support")
			}

			app = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_supply_python_app"))
			app.Buildpacks = []string{
				"https://github.com/cloudfoundry/dotnet-core-buildpack#develop",
				"python_buildpack",
			}
			app.Disk = "1G"
		})

		It("finds the supplied dependency in the runtime container", func() {
			PushAppAndConfirm(app)

			Expect(app.Stdout.String()).To(ContainSubstring("Supplying Dotnet Core"))
			Expect(app.GetBody("/")).To(MatchRegexp(`dotnet: \d+\.\d+\.\d+`))

		})
	})

	Context("an app is pushed multiple times", func() {
		var tmpDir string

		BeforeEach(func() {
			if ok, err := cutlass.ApiGreaterThan("2.65.1"); err != nil || !ok {
				Skip("API version does not have multi-buildpack support")
			}

			var err error
			tmpDir, err = cutlass.CopyFixture(filepath.Join(bpDir, "fixtures", "flask_git_req"))
			Expect(err).To(BeNil())
			app = cutlass.New(tmpDir)
		})

		AfterEach(func() {
			os.RemoveAll(tmpDir)
		})

		It("pushes successfully both times", func() {
			app.Buildpacks = []string{
				"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
				"python_buildpack",
			}
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))

			app.Buildpacks = []string{
				"https://github.com/cloudfoundry/binary-buildpack#develop",
				"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
				"python_buildpack",
			}
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Hello, World!"))
		})
	})

	Context("the app uses miniconda", func() {
		BeforeEach(func() {
			if ok, err := cutlass.ApiGreaterThan("2.65.1"); err != nil || !ok {
				Skip("API version does not have multi-buildpack support")
			}

			app = cutlass.New(filepath.Join(bpDir, "fixtures", "miniconda_python_3"))
			app.Buildpacks = []string{
				"https://buildpacks.cloudfoundry.org/fixtures/supply-cache-new.zip",
				"python_buildpack",
			}
			app.Memory = "1GB"
			app.Disk = "2GB"
		})

		It("uses miniconda", func() {
			PushAppAndConfirm(app)
			Expect(app.Stdout.String()).To(ContainSubstring("scipy"))

			body, err := app.GetBody("/")
			Expect(err).To(BeNil())

			Expect(body).To(MatchRegexp(`numpy: \d+\.\d+\.\d+`))
			Expect(body).To(MatchRegexp(`scipy: \d+\.\d+\.\d+`))
			Expect(body).To(MatchRegexp(`sklearn: \d+\.\d+\.\d+`))
			Expect(body).To(MatchRegexp(`pandas: \d+\.\d+\.\d+`))
			Expect(body).To(ContainSubstring("python-version3"))
		})
	})
})
