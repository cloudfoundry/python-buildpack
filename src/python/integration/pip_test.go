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

func testPip(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		context("when there is a requirements.txt file", func() {
			context("when using modifiers in requirements.txt", func() {
				context("using editable mode (-e)", func() {
					it.Before(func() {
						var err error
						source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
						Expect(err).NotTo(HaveOccurred())

						file, err := os.OpenFile(filepath.Join(source, "requirements.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
						Expect(err).NotTo(HaveOccurred())

						_, err = file.WriteString("-e git+https://github.com/eregs/regulations-core.git@2.0.0#egg=regcore")
						Expect(err).NotTo(HaveOccurred())

						Expect(file.Close()).To(Succeed())
					})

					it.After(func() {
						Expect(os.RemoveAll(source)).To(Succeed())
					})

					it("handles recursive requirements successfully", func() {
						deployment, logs, err := platform.Deploy.
							Execute(name, source)
						Expect(err).NotTo(HaveOccurred())

						Eventually(deployment).Should(Serve("Hello, World!"))

						Expect(logs.String()).To(SatisfyAll(
							ContainSubstring("Running Pip Install (Unvendored)"),
							ContainSubstring("Cloning https://github.com/eregs/regulations-core.git"),
						))
					})
				})

				context("using recursive mode (-r)", func() {
					it.Before(func() {
						var err error
						source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
						Expect(err).NotTo(HaveOccurred())

						Expect(os.Remove(filepath.Join(source, "requirements.txt"))).To(Succeed())

						err = os.Mkdir(filepath.Join(source, "sub_folder"), 0755)
						Expect(err).NotTo(HaveOccurred())

						CreateRequirementsTxtFile(Expect, filepath.Join(source, "sub_folder"), "other_requirement1.txt", "Markupsafe", "-r other_requirement2.txt")
						CreateRequirementsTxtFile(Expect, filepath.Join(source, "sub_folder"), "other_requirement2.txt", "Werkzeug")
						CreateRequirementsTxtFile(Expect, source, "requirements.txt", "Flask", "Jinja2", "gunicorn", "itsdangerous", "pylibmc", "cffi", "-r sub_folder/other_requirement1.txt")
					})

					it.After(func() {
						Expect(os.RemoveAll(source)).To(Succeed())
					})

					it("installs the regulations-core package", func() {
						deployment, logs, err := platform.Deploy.
							Execute(name, source)
						Expect(err).NotTo(HaveOccurred())

						Eventually(deployment).Should(Serve("Hello, World!"))

						Expect(logs.String()).To(SatisfyAll(
							ContainSubstring("Collecting Markupsafe"),
							ContainSubstring("Collecting Werkzeug"),
						))
					})
				})
			})

			context("when specifying pip version", func() {
				context("when using default version", func() {
					it.Before(func() {
						var err error
						source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
						Expect(err).NotTo(HaveOccurred())
					})

					it.After(func() {
						Expect(os.RemoveAll(source)).To(Succeed())
					})

					it("uses python's pip module", func() {
						deployment, logs, err := platform.Deploy.
							WithEnv(map[string]string{"BP_PIP_VERSION": ""}).
							Execute(name, source)
						Expect(err).NotTo(HaveOccurred())

						Eventually(deployment).Should(Serve("Hello, World!"))

						Expect(logs.String()).To(SatisfyAll(
							ContainSubstring("Using python's pip module"),
						))
					})
				})

				context("when using latest version", func() {
					it.Before(func() {
						var err error
						source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
						Expect(err).NotTo(HaveOccurred())
					})

					it.After(func() {
						Expect(os.RemoveAll(source)).To(Succeed())
					})

					it("uses latest from manifest", func() {
						deployment, logs, err := platform.Deploy.
							WithEnv(map[string]string{"BP_PIP_VERSION": "latest"}).
							Execute(name, source)
						Expect(err).NotTo(HaveOccurred())

						Eventually(deployment).Should(Serve("Hello, World!"))

						Expect(logs.String()).To(SatisfyAll(
							ContainSubstring("Installing pip"),
							MatchRegexp(`Successfully installed pip-\d+.\d+(.\d+)?`),
						))
					})
				})
			})
		})

		context("when there is no requirements.txt file", func() {
			context("when there is a setup.py file", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "setup_py"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("successfully deploys", func() {
					deployment, logs, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve("Knock Knock. Who is there?"))

					Expect(logs.String()).To(SatisfyAll(
						Not(ContainSubstring("Skipping 'pip install' since requirements.txt does not exist")),
					))
				})
			})

			context("when there is no setup.py file", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "no_deps"))
					Expect(err).NotTo(HaveOccurred())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("successfully deploys", func() {
					deployment, logs, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve("Here is your output for /"))

					Expect(logs.String()).To(SatisfyAll(
						Not(ContainSubstring("Skipping 'pip install' since requirements.txt does not exist")),
					))
				})
			})
		})
	}
}
