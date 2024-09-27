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

func testMiniforge(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect     = NewWithT(t).Expect
			Eventually = NewWithT(t).Eventually

			name string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		context("when pushing an app that uses miniforge", func() {
			var source string
			context("when environment.yml stays the same", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "miniforge"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("successfully deploys and installs the packages", func() {
					deployment, logs, err := platform.Deploy.
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(ContainSubstring("gunicorn: 20.1.0")))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Installing Miniforge"),
						ContainSubstring("Installing conda environment from environment.yml"),
					))
				})
			})

			context("when environment.yml changes", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "miniforge"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("successfully deploys and installs the packages", func() {
					deployment, logs, err := platform.Deploy.
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(SatisfyAll(
						ContainSubstring("gunicorn: 20.1.0"),
						Not(ContainSubstring("gunicorn: 20.0.4")),
					)))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Installing Miniforge"),
						ContainSubstring("Installing conda environment from environment.yml"),
					))

					file, err := os.ReadFile(filepath.Join(source, "environment.yml"))
					Expect(err).NotTo(HaveOccurred())

					output := strings.Replace(string(file), "gunicorn=20.1.0", "gunicorn=20.0.4", 1)
					err = os.WriteFile(filepath.Join(source, "environment.yml"), []byte(output), 0644)
					Expect(err).NotTo(HaveOccurred())

					deployment, logs, err = platform.Deploy.
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve(SatisfyAll(
						ContainSubstring("gunicorn: 20.0.4"),
						Not(ContainSubstring("gunicorn: 20.1.0")),
					)))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Installing Miniforge"),
						ContainSubstring("Installing conda environment from environment.yml"),
					))
				})
			})
		})
	}
}
