package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("override yml", func() {
	var app *cutlass.App
	var buildpackName string

	BeforeEach(func() {
		if !ApiHasMultiBuildpack() {
			Skip("Multi buildpack support is required")
		}

		if isSerialTest {
			Skip("Skipping parallel tests")
		}

		buildpackName = "override_yml_" + cutlass.RandStringRunes(5)
		Expect(cutlass.CreateOrUpdateBuildpack(buildpackName, Fixtures("overrideyml_bp"), "")).To(Succeed())

		app = cutlass.New(Fixtures("no_deps"))
		app.Buildpacks = []string{buildpackName + "_buildpack", "python_buildpack"}
	})

	AfterEach(func() {
		if buildpackName != "" {
			err := cutlass.DeleteBuildpack(buildpackName)
			Expect(err).ToNot(HaveOccurred())
		}

		if app != nil {
			err := app.Destroy()
			Expect(err).ToNot(HaveOccurred())
			app = nil
		}
	})

	It("Forces python from override buildpack", func() {
		Expect(app.Push()).ToNot(Succeed())

		Expect(app.Stdout.String()).To(ContainSubstring("-----> OverrideYML Buildpack"))
		Expect(app.Stdout.String()).To(ContainSubstring("-----> Python Buildpack version " + buildpackVersion))

		Expect(app.Stdout.String()).To(ContainSubstring("-----> Installing python"))
		Expect(app.Stdout.String()).To(MatchRegexp("Copy .*/python.tgz"))
		Expect(app.Stdout.String()).To(ContainSubstring("Could not install python: dependency sha256 mismatch: expected sha256 062d906c87839d03b243e2821e10653c89b4c92878bfe2bf995dec231e117bfc, actual sha256 b56b58ac21f9f42d032e1e4b8bf8b8823e69af5411caa15aee2b140bc756962f"))
	})
})
