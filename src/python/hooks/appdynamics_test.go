package hooks_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"python/hooks"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
)


func createFile(dir, filename, command string, perm os.FileMode) error {
	procFile := filepath.Join(dir, filename)
	f, err := os.OpenFile(procFile, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err = f.WriteString(command); err != nil {
		return err
	}
	return nil
}

var _ = Describe("Appdynamics", func() {
	var (
		err         error
		buildDir    string
		depsDir     string
		stager      *libbuildpack.Stager
		buffer      *bytes.Buffer
		appdynamics hooks.AppdynamicsHook
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "python-buildpack.build.")
		Expect(err).To(BeNil())

		depsDir, err = ioutil.TempDir("", "python-buildpack.deps.")
		Expect(err).To(BeNil())

		buffer = new(bytes.Buffer)
		logger := libbuildpack.NewLogger(ansicleaner.New(buffer))

		args := []string{buildDir, "", depsDir, "9"}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})

		command := &libbuildpack.Command{}

		appdynamics = hooks.AppdynamicsHook{
			Log:     logger,
			Command: command,
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(buildDir)).To(Succeed())
	})

	Describe("GenerateStartUpCommand", func() {

		It("Startup command correct format", func() {
			startCommand := "web: python flask.py"
			ModifiedCommand, err := appdynamics.GenerateStartUpCommand(startCommand)
			Expect(ModifiedCommand).To(Equal("web: pyagent run --  python flask.py"))
			Expect(err).To(BeNil())
		})

		It("Wrong format", func() {
			startCommand := "python flask.py"
			_, err := appdynamics.GenerateStartUpCommand(startCommand)
			Expect(err).ToNot(BeNil())
		})

	})

	Describe("RewriteProcFile", func() {
		var (
			err         error
			tempProcDir string
		)
		BeforeEach(func() {
			tempProcDir, err = ioutil.TempDir("", "Procfiles")

			err = createFile(tempProcDir, "Procfile", "web: python app.py", 0644)
			Expect(err).To(BeNil())

			err = createFile(tempProcDir, "NoPermProcFile", "web: python app.py", 0444)
			Expect(err).To(BeNil())

			err = createFile(tempProcDir, "WrongFormatProcFile", "python app.py", 0644)
			Expect(err).To(BeNil())

		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempProcDir)).To(Succeed())
		})

		It("Procfile doesn't exist", func() {
			err := appdynamics.RewriteProcFile("/doesnt/exist")
			Expect(err).ToNot(BeNil())
		})

		It("Procfile with no write permissions", func() {
			err := appdynamics.RewriteProcFile(filepath.Join(tempProcDir, "NoPermProcFile"))
			Expect(err).ToNot(BeNil())
		})

		It("Procfile with no wrong format", func() {
			err := appdynamics.RewriteProcFile(filepath.Join(tempProcDir, "WrongFormatProcFile"))
			Expect(err).ToNot(BeNil())
		})

		It("procfile will be rewritten with pyagent", func() {
			err := appdynamics.RewriteProcFile(filepath.Join(tempProcDir, "Procfile"))
			Expect(err).To(BeNil())
			startCommand, err := ioutil.ReadFile(filepath.Join(tempProcDir, "Procfile"))
			Expect(err).To(BeNil())
			Expect(string(startCommand)).To(Equal("web: pyagent run --  python app.py"))
		})

	})

	Describe("RewriteRequirementsFile when requirements.txt is not packaged", func() {
		It("requirements.txt doesn't exist", func() {
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).ToNot(BeTrue())
			err := appdynamics.RewriteRequirementsFile(stager)
			Expect(err).To(BeNil())
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packagesList, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).To(BeNil())
			Expect(string(packagesList)).To(Equal("appdynamics"))
		})
	})

	Describe("RewriteRequirementsFile when requirements.txt is packaged", func() {
		BeforeEach(func() {
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).ToNot(BeTrue())
			procFile := filepath.Join(buildDir, "requirements.txt")
			f, err := os.OpenFile(procFile, os.O_CREATE|os.O_WRONLY, 0644)
			Expect(err).To(BeNil())
			defer f.Close()
			_, err = f.WriteString("Flask")
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "requirements.txt"))).To(Succeed())
		})

		It("requirements.txt doesn't exist", func() {
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			err := appdynamics.RewriteRequirementsFile(stager)
			Expect(err).To(BeNil())
			packages, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(string(packages)).To(Equal("Flask\nappdynamics"))
		})
	})

	Describe("GenerateScript", func() {
		It("Generate script from Env map", func() {
			envVal := map[string]string{
				"APPD_KEY_1": "APPD_VAL_1",
				"APPD_KEY_2": "APPD_VAL_2",
			}
			script := appdynamics.GenerateAppdynamicsScript(envVal)
			expectedScript := `# Autogenerated Appdynamics Script 

export APPD_KEY_1=APPD_VAL_1
export APPD_KEY_2=APPD_VAL_2`
			Expect(script).To(Equal(expectedScript))
		})
	})

	Describe("CreateAppDynamicsEnv", func() {
		It("Generate script from Env map", func() {
			envVal := map[string]string{
				"APPD_KEY_1": "APPD_VAL_1",
				"APPD_KEY_2": "APPD_VAL_2",
			}
			appdynamics.CreateAppDynamicsEnv(stager, envVal)
			appdynamicsShellScript := filepath.Join(stager.DepDir(), "profile.d", "appdynamics.sh")
			Expect(libbuildpack.FileExists(appdynamicsShellScript)).To(BeTrue())
			expectedScript := `# Autogenerated Appdynamics Script 

export APPD_KEY_1=APPD_VAL_1
export APPD_KEY_2=APPD_VAL_2`
			script, err := ioutil.ReadFile(appdynamicsShellScript)
			Expect(err).To(BeNil())
			Expect(string(script)).To(Equal(expectedScript))
		})
	})

	Describe("BeforeCompile when VCAP_SERVICES is not present", func() {
		BeforeEach(func() {
			Expect(os.Getenv("VCAP_SERVICES")).To(Equal(""))
			err = createFile(buildDir, "Procfile", "web: python app.py", 0644)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "Procfile"))).To(Succeed())
		})

		It("VCAP_SERVICES is not present", func() {
			err := appdynamics.BeforeCompile(stager)
			Expect(err).To(BeNil())
			Expect(libbuildpack.FileExists(filepath.Join(stager.DepDir(), "profile.d", "appdynamics.sh"))).To(BeFalse())
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeFalse())
			procCommand, err := ioutil.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).To(BeNil())
			Expect(string(procCommand)).To(Equal("web: python app.py"))
		})
	})

	Describe("BeforeCompile when VCAP_SERVICES has no appdynamics", func() {
		BeforeEach(func() {
			Expect(os.Getenv("VCAP_SERVICES")).To(Equal(""))
			os.Setenv("VCAP_SERVICES", `{"service": [{"credentials": {"login": "name"}, "name": "443"}]}`)
			err = createFile(buildDir, "Procfile", "web: python app.py", 0644)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "Procfile"))).To(Succeed())
			os.Unsetenv("VCAP_SERVICES")
		})

		It("VCAP_SERVICES has no appdynamics", func() {
			err := appdynamics.BeforeCompile(stager)
			Expect(err).To(BeNil())
			Expect(libbuildpack.FileExists(filepath.Join(stager.DepDir(), "profile.d", "appdynamics.sh"))).To(BeFalse())
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeFalse())
			procCommand, err := ioutil.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).To(BeNil())
			Expect(string(procCommand)).To(Equal("web: python app.py"))
		})
	})

	Describe("BeforeCompile when VCAP_SERVICES has appdynamics", func() {
		BeforeEach(func() {
			os.Setenv("VCAP_SERVICES",
				`{"appdynamics": [{"instance_name": "plan-instance", "tags": [], "name": "plan", "syslog_drain_url": null, "binding_name": null, "credentials": {"host-name": "controller.test.com", "plan-name": "plan", "guid": "guid", "plan-description": "plan", "ssl-enabled": false, "account-access-key": "key", "account-name": "account-name", "port": "7777"}, "label": "appdynamics"}]}`)
			os.Setenv("VCAP_APPLICATION", `{"application_id": "applicationId", "name": "test",  "application_name": "test"}`)
			os.Setenv("APPD_TIER_NAME", "tier" )
			os.Setenv("APPD_NODE_NAME", "node" )

			err = createFile(buildDir, "Procfile", "web: python app.py", 0644)
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "Procfile"))).To(Succeed())
			os.Unsetenv("VCAP_SERVICES")
		})

		It("VCAP_SERVICES has appdynamics", func() {
			err := appdynamics.BeforeCompile(stager)
			Expect(err).To(BeNil())

			Expect(libbuildpack.FileExists(filepath.Join(stager.DepDir(), "profile.d", "appdynamics.sh"))).To(BeTrue())
			appdynamicsInfo, err := ioutil.ReadFile(filepath.Join(stager.DepDir(), "profile.d", "appdynamics.sh"))
			expectedInfo := `# Autogenerated Appdynamics Script 

export APPD_ACCOUNT_ACCESS_KEY=key
export APPD_ACCOUNT_NAME=account-name
export APPD_APP_NAME=test
export APPD_CONTROLLER_HOST=controller.test.com
export APPD_CONTROLLER_PORT=7777
export APPD_NODE_NAME=node
export APPD_SSL_ENABLED=off
export APPD_TIER_NAME=tier`
			Expect(string(appdynamicsInfo)).To(Equal(expectedInfo))
			Expect(err).To(BeNil())

			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packages, err := ioutil.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).To(BeNil())
			Expect(string(packages)).To(Equal("appdynamics"))

			procCommand, err := ioutil.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).To(BeNil())
			Expect(string(procCommand)).To(Equal("web: pyagent run --  python app.py"))
		})
	})

})
