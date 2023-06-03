package integration_test

import (
	"flag"
	"github.com/onsi/gomega/types"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/switchblade"
	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	settings struct {
		Buildpack struct {
			Version string
			Path    string
		}

		Cached      bool
		Serial      bool
		GitHubToken string
		Platform    string
		Stack       string
	}
)

func init() {
	flag.BoolVar(&settings.Cached, "cached", false, "run cached buildpack tests")
	flag.BoolVar(&settings.Serial, "serial", false, "run serial buildpack tests")
	flag.StringVar(&settings.Platform, "platform", "cf", `switchblade platform to test against ("cf" or "docker")`)
	flag.StringVar(&settings.GitHubToken, "github-token", "", "use the token to make GitHub API requests")
	flag.StringVar(&settings.Stack, "stack", "cflinuxfs3", "stack to use when pushing apps")
}

func TestIntegration(t *testing.T) {
	var Expect = NewWithT(t).Expect

	format.MaxLength = 0
	SetDefaultEventuallyTimeout(10 * time.Second)

	root, err := filepath.Abs("./../../..")
	Expect(err).NotTo(HaveOccurred())

	fixtures := filepath.Join(root, "fixtures")

	platform, err := switchblade.NewPlatform(settings.Platform, settings.GitHubToken, settings.Stack)
	Expect(err).NotTo(HaveOccurred())

	err = platform.Initialize(
		switchblade.Buildpack{
			Name: "python_buildpack",
			URI:  os.Getenv("BUILDPACK_FILE"),
		},
		switchblade.Buildpack{
			Name: "override_buildpack",
			URI:  filepath.Join(fixtures, "util", "override_buildpack"),
		},
	)
	Expect(err).NotTo(HaveOccurred())

	proxyName, err := switchblade.RandomName()
	Expect(err).NotTo(HaveOccurred())

	proxyDeploymentProcess := platform.Deploy.WithBuildpacks("go_buildpack")

	// TODO: remove this once go-buildpack runs on cflinuxfs4
	// This is done to have the proxy app written in go up and running
	if settings.Stack == "cflinuxfs4" {
		proxyDeploymentProcess = proxyDeploymentProcess.WithStack("cflinuxfs3")
	}

	proxyDeployment, _, err := proxyDeploymentProcess.
		Execute(proxyName, filepath.Join(fixtures, "util", "proxy"))
	Expect(err).NotTo(HaveOccurred())

	dynatraceName, err := switchblade.RandomName()
	Expect(err).NotTo(HaveOccurred())

	dynatraceDeploymentProcess := platform.Deploy.WithBuildpacks("go_buildpack")

	// TODO: remove this once go-buildpack runs on cflinuxfs4
	// This is done to have the dynatrace broker app app written in go up and running
	if settings.Stack == "cflinuxfs4" {
		dynatraceDeploymentProcess = dynatraceDeploymentProcess.WithStack("cflinuxfs3")
	}

	dynatraceDeployment, _, err := dynatraceDeploymentProcess.
		Execute(dynatraceName, filepath.Join(fixtures, "util", "dynatrace"))
	Expect(err).NotTo(HaveOccurred())

	suite := spec.New("integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault(platform, fixtures))
	suite("Dynatrace", testDynatrace(platform, fixtures, dynatraceDeployment.InternalURL))
	suite("AppDynamics", testAppDynamics(platform, fixtures))
	suite("Pip", testPip(platform, fixtures))
	suite("Versions", testVersions(platform, fixtures, root))
	suite("Miscellaneous", testMiscellaneous(platform, fixtures))
	suite("Django", testDjango(platform, fixtures))
	suite("Miniconda", testMiniconda(platform, fixtures))
	suite("Pipenv", testPipenv(platform, fixtures))
	suite("Override", testOverride(platform, fixtures))
	suite("Multibuildpack", testMultibuildpack(platform, fixtures))
	suite("Sealights", testSealights(platform, fixtures))

	if settings.Cached {
		suite("Offline", testOffline(platform, fixtures))
	} else {
		suite("Cache", testCache(platform, fixtures))
		suite("Proxy", testProxy(platform, fixtures, proxyDeployment.InternalURL))
	}

	suite.Run(t)

	Expect(platform.Delete.Execute(proxyName)).To(Succeed())
	Expect(platform.Delete.Execute(dynatraceName)).To(Succeed())
	Expect(os.Remove(os.Getenv("BUILDPACK_FILE"))).To(Succeed())
}

func CreateRequirementsTxtFile(Expect func(actual interface{}, extra ...interface{}) types.Assertion, path string, fileName string, modules ...string) {
	file, err := os.Create(filepath.Join(path, fileName))
	Expect(err).NotTo(HaveOccurred())

	for _, module := range modules {
		_, err = file.WriteString(module + "\n")
		Expect(err).NotTo(HaveOccurred())
	}

	err = file.Close()

	Expect(err).NotTo(HaveOccurred())
}
