package integration_test

import (
	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"
	"os"
	"path/filepath"
	"testing"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testMiscellaneous(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		context("when pushing an app that uses nltk corpus", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "miscellaneous", "nltk"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("deploys successfully", func() {
				deployment, logs, err := platform.Deploy.
					WithEnv(map[string]string{"BP_DEBUG": "1"}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("The Fulton County Grand Jury said Friday an investigation of Atlanta's recent primary election produced")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Downloading NLTK packages: brown"),
					ContainSubstring("Dir checksum unchanged"),
				))
			})
		})

		context("when pushing an app that uses tkinter", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "miscellaneous", "tkinter"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("deploys successfully", func() {
				deployment, _, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("tkinter was imported")))
			})
		})

		context("when pushing an app that has pre and post scripts", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "miscellaneous", "hooks"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("deploys successfully and runs the scripts", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("Echo from app pre compile"),
					ContainSubstring("Echo from app post compile"),
				))
			})
		})

		context("when pushing an app without a Procfile", func() {
			context("when start command is specified in push", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
					Expect(err).NotTo(HaveOccurred())

					err = os.Remove(filepath.Join(source, "Procfile"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("deploys successfully", func() {
					deployment, logs, err := platform.Deploy.
						WithEnv(map[string]string{"BP_DEBUG": "1"}).
						WithStartCommand("gunicorn server:app").
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())
					Eventually(deployment).Should(Serve(ContainSubstring("Hello, World!")))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("No start command specified by buildpack or via Procfile"),
						ContainSubstring("App will not start unless a command is provided at runtime"),
					))
				})
			})

			context("when start command is not specified in manifest.yml", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
					Expect(err).NotTo(HaveOccurred())

					err = os.Remove(filepath.Join(source, "Procfile"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("displays a nice error messages and gracefully fails", func() {
					_, logs, err := platform.Deploy.
						Execute(name, source)
					Expect(err).To(HaveOccurred())

					Expect(err.Error()).To(ContainSubstring("error: Start command not specified"))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("No start command specified by buildpack or via Procfile"),
						ContainSubstring("App will not start unless a command is provided at runtime"),
					))
				})
			})
		})
	}
}
