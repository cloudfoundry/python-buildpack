package integration_test

import (
	"github.com/blang/semver"
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testVersions(platform switchblade.Platform, fixtures, root string) func(*testing.T, spec.G, spec.S) {
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

		context("when there is a runtime.txt file", func() {
			var source string
			context("with an unsupported version", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
					Expect(err).NotTo(HaveOccurred())

					file, err := os.OpenFile(filepath.Join(source, "runtime.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
					Expect(err).NotTo(HaveOccurred())

					_, err = file.WriteString("python-99.99.99")
					Expect(err).NotTo(HaveOccurred())

					Expect(file.Close()).To(Succeed())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("displays a nice error messages and gracefully fails", func() {
					_, logs, err := platform.Deploy.
						Execute(name, source)
					Expect(err).To(HaveOccurred())

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Could not install python: no match found for 99.99.99"),
					))
				})
			})

			context("with a supported version", func() {
				it.Before(func() {
					var err error
					source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
					Expect(err).NotTo(HaveOccurred())

					file, err := os.OpenFile(filepath.Join(source, "runtime.txt"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
					Expect(err).NotTo(HaveOccurred())

					_, err = file.WriteString("python-3.10.x")
					Expect(err).NotTo(HaveOccurred())

					Expect(file.Close()).To(Succeed())
				})

				it.After(func() {
					Expect(os.RemoveAll(source)).To(Succeed())
				})

				it("deploy successfully", func() {
					deployment, logs, err := platform.Deploy.
						Execute(name, source)
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(Serve("Hello, World!"))

					Expect(logs.String()).To(SatisfyAll(
						ContainSubstring("Installing python 3.10"),
					))
				})
			})
		})

		context("when there is no runtime.txt file", func() {
			var source string
			var defaultV string

			type manifestContent struct {
				DefaultVersions []struct {
					Name    string `yaml:"name"`
					Version string `yaml:"version"`
				} `yaml:"default_versions"`
			}

			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
				Expect(err).NotTo(HaveOccurred())

				mc := manifestContent{}
				err = libbuildpack.NewYAML().Load(filepath.Join(root, "manifest.yml"), &mc)
				Expect(err).To(BeNil())
				for _, defaultDep := range mc.DefaultVersions {
					if defaultDep.Name == "python" {
						defaultV = defaultDep.Version
					}
				}
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("deploys with default Python version", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve("Hello, World!"))

				re := regexp.MustCompile("Installing python (.*)[\r\n|\r|\n]")
				match := re.FindStringSubmatch(logs.String())
				foundVersion := match[1]

				versionRange := semver.MustParseRange("<=" + defaultV)
				v1 := semver.MustParse(foundVersion)
				Expect(versionRange(v1)).To(BeTrue())
			})
		})
	}
}
