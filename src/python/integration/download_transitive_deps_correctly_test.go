package integration_test

import (
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("download transitive dependencies correctly", func() {
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

	Context("find-links is set for transitive dependencies", func() {
		BeforeEach(func() {
			app = cutlass.New(Fixtures("vendored_transitive_dependencies"))
			app.SetEnv("BP_DEBUG", "1")
		})

		It("should work", func() {
			PushAppAndConfirm(app)
			Expect(app.GetBody("/")).To(ContainSubstring("Welcome to Python on Cloud Foundry"))
		})

		AssertNoInternetTraffic("vendored_transitive_dependencies")
	})

})
