package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

const (
	Regexp         = `\[.*\/python[\-_][\d.]+[\-_]linux[\-_](amd64)?(x64)?[\-_]cflinuxfs\d[\-_][\da-f]+\.tgz\]`
	DownloadRegexp = "Download " + Regexp
	CopyRegexp     = "Copy " + Regexp
)

func testCache(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
	return func(t *testing.T, context spec.G, it spec.S) {
		var (
			Expect = NewWithT(t).Expect

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = switchblade.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = switchblade.Source(filepath.Join(fixtures, "simple"))
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		it("uses the cache for manifest dependencies", func() {
			deploy := platform.Deploy.
				WithBuildpacks("python_buildpack")

			_, logs, err := deploy.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(MatchRegexp(DownloadRegexp)))
			Expect(logs).NotTo(ContainLines(MatchRegexp(CopyRegexp)))

			_, logs, err = deploy.Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).NotTo(ContainLines(MatchRegexp(DownloadRegexp)))
			Expect(logs).To(ContainLines(MatchRegexp(CopyRegexp)))
		})
	}
}
