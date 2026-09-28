package integration_test

import (
	"os"
	"path/filepath"
	"strings"
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
					ContainSubstring("Running uv sync"),
					ContainSubstring("+ flask==3.1.3"),
					ContainSubstring("+ gunicorn==26.2.0"),
					Not(ContainSubstring("pytest")),
					Not(ContainSubstring("Running Pip Install")),
				))
			})
		})

		context("when pushing an app whose uv.lock contains a wrong hash", func() {
			it.Before(func() {
				lockPath := filepath.Join(source, "uv.lock")
				lock, err := os.ReadFile(lockPath)
				Expect(err).NotTo(HaveOccurred())

				const blinkerWheelHash = "sha256:ba0efaa9080b619ff2f3459d1d500c57bddea4a6b424b60a91141db6fd2f08bc"
				Expect(string(lock)).To(ContainSubstring(blinkerWheelHash))
				lock = []byte(strings.ReplaceAll(string(lock), blinkerWheelHash, "sha256:"+strings.Repeat("0", 64)))
				Expect(os.WriteFile(lockPath, lock, 0644)).To(Succeed())
			})

			it("fails to stage", func() {
				_, logs, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					Execute(name, source)
				Expect(err).To(HaveOccurred())

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Hash mismatch for `blinker==1.9.0`"),
					ContainSubstring("Could not install packages with uv"),
				))
			})
		})

		context("when pushing a Django app that has a uv.lock", func() {
			it.Before(func() {
				Expect(os.RemoveAll(source)).To(Succeed())

				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "uv_django"))
				Expect(err).NotTo(HaveOccurred())
			})

			it("installs the packages with uv and runs collectstatic", func() {
				deployment, logs, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("The install worked successfully!")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Running uv sync"),
					ContainSubstring("+ django==4.1.5"),
					ContainSubstring("collectstatic --noinput"),
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
