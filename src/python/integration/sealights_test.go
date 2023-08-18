package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testSealights(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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
			source, err = switchblade.Source(filepath.Join(fixtures, "services", "sealights"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		it("builds the app with a Sealights agent", func() {
			deployment, logs, err := platform.Deploy.
				WithBuildpacks("python_buildpack").
				WithServices(map[string]switchblade.Service{
					"sealights": {
						"tokenFile": "sltoken.txt",
					},
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())
			Eventually(deployment).Should(SatisfyAll(
				Serve(ContainSubstring("OK")).WithEndpoint("/health"),
			))
			Expect(logs.String()).To(SatisfyAll(
				ContainSubstring("Setting up Sealights hook"),
				ContainSubstring("Rewriting ProcFile to start with Sealights"),
				ContainSubstring("Successfully set up Sealights hook"),
			))
		})
	}
}
