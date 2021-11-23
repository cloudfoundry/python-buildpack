package integration_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var bpDir string
var buildpackVersion string
var isSerialTest bool
var packagedBuildpack cutlass.VersionedBuildpackPackage
var token string
var platform string

var _ = func() bool {
	testing.Init()
	return true
}()

func init() {
	flag.StringVar(&buildpackVersion, "version", "", "version to use (builds if empty)")
	flag.BoolVar(&cutlass.Cached, "cached", true, "cached buildpack")
	flag.BoolVar(&isSerialTest, "serial", false, "run just serial tests")
	flag.StringVar(&cutlass.DefaultMemory, "memory", "128M", "default memory for pushed apps")
	flag.StringVar(&cutlass.DefaultDisk, "disk", "384M", "default disk for pushed apps")
	flag.StringVar(&token, "github-token", "", "use the token to make GitHub API requests")
	flag.StringVar(&platform, "platform", "cf", "platform to run tests against")
	flag.Parse()
}

var _ = SynchronizedBeforeSuite(func() []byte {
	// Run once
	if buildpackVersion == "" {
		packagedBuildpack, err := cutlass.PackageUniquelyVersionedBuildpack(os.Getenv("CF_STACK"), ApiHasStackAssociation())
		Expect(err).NotTo(HaveOccurred())

		data, err := json.Marshal(packagedBuildpack)
		Expect(err).NotTo(HaveOccurred())
		return data
	}

	return []byte{}
}, func(data []byte) {
	// Run on all nodes
	var err error
	if len(data) > 0 {
		err = json.Unmarshal(data, &packagedBuildpack)
		Expect(err).NotTo(HaveOccurred())
		buildpackVersion = packagedBuildpack.Version
	}

	bpDir, err = cutlass.FindRoot()
	Expect(err).NotTo(HaveOccurred())

	Expect(cutlass.CopyCfHome()).To(Succeed())
	cutlass.SeedRandom()
	cutlass.DefaultStdoutStderr = GinkgoWriter

	SetDefaultEventuallyTimeout(10 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {
	// Run on all nodes
}, func() {
	// Run once
	Expect(cutlass.RemovePackagedBuildpack(packagedBuildpack)).To(Succeed())
	Expect(cutlass.DeleteOrphanedRoutes()).To(Succeed())
})

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

func PushAppAndConfirm(app *cutlass.App) {
	Expect(app.Push()).To(Succeed())
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
	Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
}

func Restart(app *cutlass.App) {
	Expect(app.Restart()).To(Succeed())
	Eventually(func() ([]string, error) { return app.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))
}

func ApiHasTask() bool {
	supported, err := cutlass.ApiGreaterThan("2.75.0")
	Expect(err).NotTo(HaveOccurred())
	return supported
}

func ApiHasMultiBuildpack() bool {
	supported, err := cutlass.ApiGreaterThan("2.90.0")
	Expect(err).NotTo(HaveOccurred())
	return supported
}

func ApiHasStackAssociation() bool {
	supported, err := cutlass.ApiGreaterThan("2.113.0")
	Expect(err).NotTo(HaveOccurred())
	return supported
}

func Fixtures(names ...string) string {
	root, err := cutlass.FindRoot()
	Expect(err).NotTo(HaveOccurred())

	names = append([]string{root, "fixtures"}, names...)
	return filepath.Join(names...)
}

func AssertUsesProxyDuringStagingIfPresent(fixtureName string) {
	Context("with an uncached buildpack", func() {
		BeforeEach(func() {
			if cutlass.Cached {
				Skip("Running cached tests")
			}
		})

		It("uses a proxy during staging if present", func() {
			proxy, err := cutlass.NewProxy()
			Expect(err).To(BeNil())
			defer proxy.Close()

			bpFile := filepath.Join(bpDir, buildpackVersion+"tmp")
			cmd := exec.Command("cp", packagedBuildpack.File, bpFile)
			err = cmd.Run()
			Expect(err).To(BeNil())
			defer os.Remove(bpFile)

			traffic, _, _, err := cutlass.InternetTraffic(
				Fixtures(fixtureName),
				bpFile,
				[]string{"HTTP_PROXY=" + proxy.URL, "HTTPS_PROXY=" + proxy.URL},
			)
			Expect(err).To(BeNil())
			// Expect(built).To(BeTrue())

			destUrl, err := url.Parse(proxy.URL)
			Expect(err).To(BeNil())

			Expect(cutlass.UniqueDestination(
				traffic, fmt.Sprintf("%s.%s", destUrl.Hostname(), destUrl.Port()),
			)).To(BeNil())
		})
	})
}

func AssertNoInternetTraffic(fixtureName string) {
	It("has no traffic", func() {
		if !cutlass.Cached {
			Skip("Running uncached tests")
		}

		bpFile := filepath.Join(bpDir, buildpackVersion+"tmp")
		cmd := exec.Command("cp", packagedBuildpack.File, bpFile)
		err := cmd.Run()
		Expect(err).To(BeNil())
		defer os.Remove(bpFile)

		traffic, built, logs, err := cutlass.InternetTraffic(
			Fixtures(fixtureName),
			bpFile,
			[]string{"LC_ALL C.UTF-8", "LANG C.UTF-8"},
		)
		Expect(err).To(BeNil())
		Expect(built).To(BeTrue(), strings.Join(logs, "\n"))
		Expect(traffic).To(BeEmpty(), strings.Join(logs, "\n"))
	})
}

func RunCf(args ...string) error {
	command := exec.Command("cf", args...)
	command.Stdout = GinkgoWriter
	command.Stderr = GinkgoWriter
	return command.Run()
}
