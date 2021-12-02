package integration_test

import (
	"os/exec"

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
			cutlass.DeleteBuildpack(buildpackName)
		}

		if app != nil {
			app.Destroy()
			app = nil
		}
	})

	It("Forces python from override buildpack", func() {
		Expect(app.Push()).ToNot(Succeed())

		logs := exec.Command("cf", "logs", "--recent", app.Name)
		out, err := logs.CombinedOutput()
		Expect(err).ToNot(HaveOccurred())

		Expect(out).To(ContainSubstring("-----> OverrideYML Buildpack"))
		Expect(out).To(ContainSubstring("-----> Python Buildpack version "+buildpackVersion))

		Expect(out).To(ContainSubstring("-----> Installing python"))
		Expect(out).To(MatchRegexp("Copy .*/python.tgz"))
		Expect(out).To(ContainSubstring("Could not install python: dependency sha256 mismatch: expected sha256 062d906c87839d03b243e2821e10653c89b4c92878bfe2bf995dec231e117bfc, actual sha256 b56b58ac21f9f42d032e1e4b8bf8b8823e69af5411caa15aee2b140bc756962f"))
	})
})
