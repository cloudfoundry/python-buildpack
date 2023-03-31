package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testOffline(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		context("when pushing an app with pip", func() {
			context("when vendor directory is complete", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "vendored", "simple"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("deploys successfully without internet access", func() {
					deployment, logs, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						WithoutInternetAccess().
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve("Hello, World!"))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Running Pip Install (Vendored)"),
						ContainSubstring("Using the pip --no-build-isolation flag since it is available"),
					))
				})
			})

			context("when vendor directory is incomplete", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "vendored", "simple"))
					Expect(err).NotTo(HaveOccurred())

					files, err := filepath.Glob(filepath.Join(source, "vendor", "*.whl"))
					Expect(err).NotTo(HaveOccurred())

					err = os.Remove(filepath.Join(source, "vendor", filepath.Base(files[0])))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("displays a nice error messages and gracefully fails", func() {
					_, logs, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						WithoutInternetAccess().
						Execute(name, source)
					Expect(err).To(HaveOccurred())

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Running Pip Install (Vendored)"),
						ContainSubstring("Running pip install failed. You need to include all dependencies in the vendor directory."),
					))

				})
			})
		})

		context("when pushing an app with pipenv", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "vendored", "pipenv"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("deploys successfully without internet access", func() {
				deployment, logs, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Generating 'requirements.txt' from Pipfile.lock"),
					ContainSubstring("Running Pip Install (Vendored)"),
				))
			})
		})
	}
}
