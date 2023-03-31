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

func testDjango(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

		context("when pushing an app django", func() {
			it.Before(func() {
				var err error
				source, err = switchblade.Source(filepath.Join(fixtures, "django"))
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(source)).To(Succeed())
			})

			it("deploys successfully", func() {
				deployment, logs, err := platform.Deploy.
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(Serve(ContainSubstring("The install worked successfully!")))

				Expect(logs.String()).To(SatisfyAll(
					ContainSubstring("collectstatic --noinput"),
					Not(MatchRegexp(`WARNING: You are using pip version \d+.\d+.\d+; however, version \d+.\d+.\d+ is available.`)),
				))
			})
		})
	}
}
