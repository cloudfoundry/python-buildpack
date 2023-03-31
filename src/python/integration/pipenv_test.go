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

func testPipenv(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		context("when pushing an app that has a Pipfile", func() {
			context("app has Pipfile.lock and no requirements.txt", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "pipenv"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("successfully deploys, generates a requirements.txt, and installs the packages", func() {
					deployment, logs, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Generating 'requirements.txt' from Pipfile.lock"),
						ContainSubstring("Running Pip Install (Unvendored)"),
					))
				})
			})

			context("app has Pipfile.lock and requirements.txt", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "pipenv"))
					Expect(err).NotTo(HaveOccurred())

					CreateRequirementsTxtFile(Expect, source, "requirements.txt",
						"Flask", "Jinja2", "MarkupSafe", "Werkzeug", "gunicorn", "itsdangerous")
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("successfully deploys and installs the packages", func() {
					deployment, logs, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

					Expect(logs.String()).To(SatisfyAll(
						Not(ContainSubstring("Installing pipenv")),
						ContainSubstring("Collecting Flask"),
						ContainSubstring("Collecting Jinja2"),
						ContainSubstring("Collecting MarkupSafe"),
						ContainSubstring("Collecting Werkzeug"),
						ContainSubstring("Collecting gunicorn"),
						ContainSubstring("Collecting itsdangerous"),
					))
				})
			})
		})

		context("when pushing an app that does not have a Pipfile", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("successfully deploys and installs the packages", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

				Expect(logs.String()).To(SatisfyAll(
					Not(ContainSubstring("Installing pipenv")),
				))
			})
		})
	}
}
