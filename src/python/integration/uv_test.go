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

func testUv(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

			source, err = switchblade.Source(filepath.Join(fixtures, "uv"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		context("when pushing an app that has a uv.lock and no requirements.txt", func() {
			it("successfully deploys, generates a requirements.txt, and installs the packages", func() {
				deployment, logs, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Using Python version 3.12 from .python-version"),
					MatchRegexp(`Installing python 3\.12\.\d+`),
					ContainSubstring("Installing uv"),
					ContainSubstring("Generating 'requirements.txt' from uv.lock"),
					ContainSubstring("Running Pip Install (Unvendored)"),
					ContainSubstring("Collecting flask==3.1.3"),
					ContainSubstring("Collecting gunicorn==26.2.0"),
					Not(ContainSubstring("pytest")),
				))
			})
		})

		context("when pushing an app that has a uv.lock and a requirements.txt", func() {
			it.Before(func() {
				CreateRequirementsTxtFile(Expect, source, "requirements.txt",
					"Flask", "Jinja2", "MarkupSafe", "Werkzeug", "gunicorn", "itsdangerous")
			})

			it("installs the packages from requirements.txt", func() {
				deployment, logs, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Found requirements.txt, ignoring uv.lock"),
					Not(ContainSubstring("Installing uv")),
					ContainSubstring("Collecting Flask"),
				))
			})
		})
	}
}
