package integration_test

import (
	"time"

	"github.com/cloudfoundry/libbuildpack/cutlass"

	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("appdynamics", func() {
	var (
		app, serviceBrokerApp                          *cutlass.App
		serviceBrokerURL, serviceName, serviceOffering string
	)

	BeforeEach(func() {
		if isSerialTest {
			Skip("Skipping parallel tests")
		}

		serviceOffering = "appdynamics-" + cutlass.RandStringRunes(20)
		serviceName = "appdynamics-" + cutlass.RandStringRunes(20)

		serviceBrokerApp = cutlass.New(Fixtures("fake_appd_service_broker"))
		serviceBrokerApp.SetEnv("OFFERING_NAME", serviceOffering)
		Expect(serviceBrokerApp.Push()).To(Succeed())
		Eventually(func() ([]string, error) { return serviceBrokerApp.InstanceStates() }, 20*time.Second).Should(Equal([]string{"RUNNING"}))

		var err error
		serviceBrokerURL, err = serviceBrokerApp.GetUrl("")
		Expect(err).ToNot(HaveOccurred())

		Expect(RunCf("create-service-broker", serviceBrokerApp.Name, "username", "password", serviceBrokerURL, "--space-scoped")).To(Succeed())
		Expect(RunCf("create-service", serviceOffering, "public", serviceName)).To(Succeed())

		app = cutlass.New(Fixtures("with_appdynamics"))
		app.Disk = "1G"
		app.SetEnv("BP_DEBUG", "true")
		PushAppAndConfirm(app)
	})

	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil

		RunCf("purge-service-offering", "-f", serviceOffering)
		RunCf("delete-service-broker", "-f", serviceBrokerApp.Name)

		if serviceBrokerApp != nil {
			serviceBrokerApp.Destroy()
		}
		serviceBrokerApp = nil
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
		var vcapServicesEnvUnmarshalled interface{}
		json.Unmarshal(([]byte)(vcapServicesEnv), &vcapServicesEnvUnmarshalled)

		appDynamicsJson := vcapServicesEnvUnmarshalled.(map[string]interface{})[serviceOffering].([]interface{})[0]
		Expect(appDynamicsJson).To(HaveKeyWithValue("credentials", map[string]interface{}{
			"account-access-key": "test-key",
			"account-name":       "test-account",
			"host-name":          "test-sb-host",
			"port":               "1234",
			"ssl-enabled":        true,
		}))
		Expect(appDynamicsJson).To(HaveKeyWithValue("label", serviceOffering))
		Expect(appDynamicsJson).To(HaveKeyWithValue("name", serviceName))

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
