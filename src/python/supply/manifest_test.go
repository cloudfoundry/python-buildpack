package supply_test

import (
	"io"
	"testing"
	"time"

	"github.com/cloudfoundry/libbuildpack"
	. "github.com/onsi/gomega"
)

func TestManifestStacks(t *testing.T) {
	Expect := NewWithT(t).Expect
	manifest, err := libbuildpack.NewManifest("../../..", libbuildpack.NewLogger(io.Discard), time.Now())
	Expect(err).NotTo(HaveOccurred())

	for _, entry := range manifest.ManifestEntries {
		Expect(entry.CFStacks).NotTo(BeEmpty())
		for _, stack := range entry.CFStacks {
			Expect(stack).To(BeElementOf("cflinuxfs4", "cflinuxfs5"))
		}
	}

	for _, stack := range []string{"cflinuxfs4", "cflinuxfs5"} {
		t.Run(stack, func(t *testing.T) {
			Expect := NewWithT(t).Expect
			t.Setenv("CF_STACK", stack)

			dependency, err := manifest.DefaultVersion("python")
			Expect(err).NotTo(HaveOccurred())
			Expect(dependency.Version).To(HavePrefix("3.10."))
		})
	}
}
