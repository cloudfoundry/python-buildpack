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

	BeforeEach(func() {
		app = cutlass.New(filepath.Join(bpDir, "cf_spec", "fixtures", "flask_no_procfile"))
	})

	It("start command is specified in manifest.yml", func() {
		app.Push()
		Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())

		Expect(app.GetBody("/")).To(ContainSubstring("I was started without a Procfile"))
	})
})
