package integration_test

import (
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

var _ = Describe("appdynamics", func() {
	var app, appdServiceBrokerApp *cutlass.App
	var sbUrl string
	const serviceName = "appdynamics"
	cfUsername := getEnv("CF_USER_NAME", "username")
	cfPassword := getEnv("CF_PASSWORD", "password")

	RunCf := func(args ...string) error {
		command := exec.Command("cf", args...)
		command.Stdout = GinkgoWriter
		command.Stderr = GinkgoWriter
		return command.Run()
	}

	BeforeEach(func() {
		appdServiceBrokerApp = cutlass.New(filepath.Join(bpDir, "fixtures", "fake_appd_service_broker"))
		Expect(appdServiceBrokerApp.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return appdServiceBrokerApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		var err error
		sbUrl, err = appdServiceBrokerApp.GetUrl("")
		Expect(err).ToNot(HaveOccurred())

		Expect(RunCf("create-service-broker", serviceName, cfUsername, cfPassword, sbUrl, "--space-scoped")).To(Succeed())
		Expect(RunCf("create-service", serviceName, "public", serviceName)).To(Succeed())

		app = cutlass.New(filepath.Join(bpDir, "fixtures", "with_appdynamics"))
		app.SetEnv("BP_DEBUG", "true")
		PushAppAndConfirm(app)
	})

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil

		RunCf("purge-service-offering", "-f", serviceName)
		RunCf("delete-service-broker", "-f", serviceName)

		if appdServiceBrokerApp != nil {
			appdServiceBrokerApp.Destroy()
		}
		appdServiceBrokerApp = nil
	})

	It("test if appdynamics was successfully bound", func() {
		By("Binding appdynamics service to the test application")
		Expect(RunCf("bind-service", app.Name, serviceName)).To(Succeed())

		By("Restaging the test application")
		app.Stdout.Reset()
		Expect(RunCf("restage", app.Name)).To(Succeed())

		By("checking if the application has started fine and has correctly bound to appdynamics")
		vcapServicesEnv, err := app.GetBody("/vcap")
		Expect(err).To(BeNil())
		vcapServicesExpected := `{"appdynamics":[{
  "name": "appdynamics",
  "instance_name": "appdynamics",
  "binding_name": null,
  "credentials": {
    "account-access-key": "test-key",
    "account-name": "test-account",
    "host-name": "test-sb-host",
    "port": "1234",
    "ssl-enabled": true
  },
  "syslog_drain_url": null,
  "volume_mounts": [

  ],
  "label": "appdynamics",
  "provider": null,
  "plan": "public",
  "tags": [

  ]
}]}`
		Expect(vcapServicesEnv).To(Equal(vcapServicesExpected))

		By("Checking if the build pack installed and started appdynamics")
		logs := app.Stdout.String()

		Expect(app.ConfirmBuildpack(buildpackVersion)).To(Succeed())
		Expect(logs).To(ContainSubstring("-----> Setting up Appdynamics"))
		Expect(logs).To(ContainSubstring("-----> Rewriting Requirements file with appdynamics package"))
		Expect(logs).To(ContainSubstring("-----> Writing Appdynamics Environment"))
		Expect(logs).To(ContainSubstring("appdynamics.proxy.watchdog"))
		Expect(logs).To(ContainSubstring("Started proxy with pid"))

		By("Checking if the buildpack properly set the APPD environment variables in apps environments")
		appEnv, err := app.GetBody("/appd")
		Expect(err).To(BeNil())
		expectedAppEnv := fmt.Sprintf(`{
  "APPD_ACCOUNT_ACCESS_KEY": "test-key",
  "APPD_ACCOUNT_NAME": "test-account",
  "APPD_APP_NAME": "%s",
  "APPD_CONTROLLER_HOST": "test-sb-host",
  "APPD_CONTROLLER_PORT": "1234",
  "APPD_NODE_NAME": "%s",
  "APPD_SSL_ENABLED": "on",
  "APPD_TIER_NAME": "%s"
}`, app.Name, app.Name, app.Name)
		Expect(appEnv).To(Equal(expectedAppEnv))

		By("unbinding the service")
		Expect(RunCf("unbind-service", app.Name, serviceName)).To(Succeed())

	})

})
