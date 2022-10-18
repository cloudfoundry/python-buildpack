package brats_test

import (
	"github.com/cloudfoundry/libbuildpack/bratshelper"
	"github.com/cloudfoundry/libbuildpack/cutlass"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/bcrypt"
)

var _ = Describe("Python buildpack", func() {
	bratshelper.UnbuiltBuildpack("python", CopyBrats)
	bratshelper.DeployingAnAppWithAnUpdatedVersionOfTheSameBuildpack(CopyBrats)
	bratshelper.StagingWithBuildpackThatSetsEOL("python", CopyBrats)
	bratshelper.StagingWithCustomBuildpackWithCredentialsInDependencies(CopyBrats)
	bratshelper.DeployAppWithExecutableProfileScript("python", CopyBrats)
	bratshelper.DeployAnAppWithSensitiveEnvironmentVariables(CopyBrats)

	bratshelper.ForAllSupportedVersions("python", CopyBrats, func(pythonVersion string, app *cutlass.App) {
		PushApp(app)

		By("runs a simple webserver", func() {
			Expect(app.GetBody("/")).To(ContainSubstring("Hello World!"))
		})
		By("uses the correct python version", func() {
			Expect(app.Stdout.String()).To(ContainSubstring("Installing python " + pythonVersion))
			Expect(app.GetBody("/version")).To(ContainSubstring(pythonVersion))
		})
		By("encrypts with bcrypt", func() {
			hashedPassword, err := app.GetBody("/bcrypt")
			Expect(err).ToNot(HaveOccurred())
			Expect(bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte("Hello, bcrypt"))).ToNot(HaveOccurred())
		})
		By("supports postgres by raising a no connection error", func() {
			Expect(app.GetBody("/pg")).To(
				Or(
					// cflinuxfs3 and cflinuxfs4 have different messages for the same error.
					ContainSubstring("could not connect to server: No such file or directory"),
					ContainSubstring("failed: No such file or directory"),
				),
			)
		})
		By("supports mysql by raising a no connection error", func() {
			Expect(app.GetBody("/mysql")).To(
				Or(
					// cflinuxfs3 and cflinuxfs4 have different messages for the same error.
					ContainSubstring("Can't connect to local MySQL server through socket"),
					ContainSubstring("Can't connect to local server through socket"),
				),
			)
		})
		By("supports loading and running the hiredis lib", func() {
			Expect(app.GetBody("/redis")).To(ContainSubstring("Hello"))
		})
		By("supports the proper version of unicode", func() {
			maxUnicode := "1114111"
			Expect(app.GetBody("/unicode")).To(ContainSubstring("max unicode: " + maxUnicode))
		})
	})
})
