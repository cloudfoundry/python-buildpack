package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/switchblade"
	"github.com/sclevine/spec"

	. "github.com/cloudfoundry/switchblade/matchers"
	. "github.com/onsi/gomega"
)

func testAppDynamics(platform switchblade.Platform, fixtures string) func(*testing.T, spec.G, spec.S) {
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

			source, err = switchblade.Source(filepath.Join(fixtures, "services", "appdynamics"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(platform.Delete.Execute(name)).To(Succeed())
		})

		for _, n := range []string{"appdynamics", "app-dynamics"} {
			service := "some-" + n

			context(fmt.Sprintf("with a service called %s", name), func() {
				it("ensures the service can be bound to the app", func() {
					deployment, _, err := platform.Deploy.
						WithBuildpacks("python_buildpack").
						WithServices(map[string]switchblade.Service{
							service: {
								"account-access-key": "test-key",
								"account-name":       "test-account",
								"host-name":          "test-ups-host",
								"port":               "1234",
								"ssl-enabled":        true,
							},
						}).
						Execute(name, filepath.Join(fixtures, "services", "appdynamics"))
					Expect(err).NotTo(HaveOccurred())

					Eventually(deployment).Should(SatisfyAll(
						Serve(ContainSubstring(`"APPD_ACCOUNT_ACCESS_KEY": "test-key"`)).WithEndpoint("/appd"),
						Serve(ContainSubstring(`"APPD_ACCOUNT_NAME": "test-account"`)).WithEndpoint("/appd"),
						Serve(ContainSubstring(`"APPD_CONTROLLER_HOST": "test-ups-host"`)).WithEndpoint("/appd"),
						Serve(ContainSubstring(`"APPD_CONTROLLER_PORT": "1234"`)).WithEndpoint("/appd"),
						Serve(ContainSubstring(`"APPD_SSL_ENABLED": "on"`)).WithEndpoint("/appd"),
					))

					response, err := http.Get(fmt.Sprintf("%s/logs", deployment.ExternalURL))
					Expect(err).NotTo(HaveOccurred())
					defer response.Body.Close()

					logs, err := io.ReadAll(response.Body)
					Expect(err).NotTo(HaveOccurred())

					Expect(string(logs)).To(SatisfyAll(
						ContainSubstring("Started proxy with pid"),
						ContainSubstring("Started watchdog with pid"),
					))
				})
			})
		}

		context("when APPD_APP_NAME, APPD_TIER_NAME and APPD_NODE_NAME are set", func() {
			it("uses the values", func() {
				deployment, _, err := platform.Deploy.
					WithBuildpacks("python_buildpack").
					WithServices(map[string]switchblade.Service{
						"appdynamics": {
							"account-access-key": "test-key",
							"account-name":       "test-account",
							"host-name":          "test-ups-host",
							"port":               "1234",
							"ssl-enabled":        true,
						},
					}).
					WithEnv(map[string]string{
						"APPD_APP_NAME":  "set-name",
						"APPD_TIER_NAME": "set-tier",
						"APPD_NODE_NAME": "set-node",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				Eventually(deployment).Should(SatisfyAll(
					Serve(ContainSubstring(`"APPD_ACCOUNT_ACCESS_KEY": "test-key"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_ACCOUNT_NAME": "test-account"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_CONTROLLER_HOST": "test-ups-host"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_CONTROLLER_PORT": "1234"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_SSL_ENABLED": "on"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_APP_NAME": "set-name"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_TIER_NAME": "set-tier"`)).WithEndpoint("/appd"),
					Serve(ContainSubstring(`"APPD_NODE_NAME": "set-node"`)).WithEndpoint("/appd"),
				))

				response, err := http.Get(fmt.Sprintf("%s/logs", deployment.ExternalURL))
				Expect(err).NotTo(HaveOccurred())
				defer response.Body.Close()

				logs, err := io.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(logs)).To(SatisfyAll(
					ContainSubstring("Started proxy with pid"),
					ContainSubstring("Started watchdog with pid"),
				))
			})
		})
	}
}
