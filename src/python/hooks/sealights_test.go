package hooks_test

import (
	"bytes"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/cloudfoundry/libbuildpack/ansicleaner"
	"github.com/cloudfoundry/python-buildpack/src/python/hooks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
)

var _ = Describe("Sealights", func() {
	var (
		err       error
		buildDir  string
		depsDir   string
		stager    *libbuildpack.Stager
		buffer    *bytes.Buffer
		sealights hooks.SealightsHook
	)
	BeforeEach(func() {
		buildDir, err = os.MkdirTemp("", "python-buildpack.build.")
		Expect(err).NotTo(HaveOccurred())

		depsDir, err = os.MkdirTemp("", "python-buildpack.deps.")
		Expect(err).NotTo(HaveOccurred())

		buffer = new(bytes.Buffer)
		logger := libbuildpack.NewLogger(ansicleaner.New(buffer))

		args := []string{buildDir, "", depsDir, buildOrder}
		stager = libbuildpack.NewStager(args, logger, &libbuildpack.Manifest{})
		sealights = hooks.SealightsHook{
			Log: logger,
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(buildDir)).To(Succeed())
		Expect(os.RemoveAll(depsDir)).To(Succeed())
	})

	Context("GenerateStartUpCommand", func() {
		It("Returns the command when it is provided in the correct format and empty config", func() {
			slConfig := hooks.NewSealightsConfig()
			startCommand := "web: python flask.py"
			ModifiedCommand, err := sealights.GenerateStartUpCommand(startCommand, slConfig.GetStartFlags())
			Expect(ModifiedCommand).To(Equal("web: sl-python run --  python flask.py"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("Returns the command when it is provided in the correct format with sl config", func() {
			slConfig := &hooks.SealightsConfig{
				Token:     "some-token",
				TokenFile: "",
				BsId:      "some-bsid",
				BsIdFile:  "",
				Proxy:     "some-proxy",
				LabId:     "some-labid",
			}
			startCommand := "web: python flask.py"
			ModifiedCommand, err := sealights.GenerateStartUpCommand(startCommand, slConfig.GetStartFlags())
			Expect(ModifiedCommand).To(Equal("web: sl-python run --token some-token --buildsessionid some-bsid --proxy some-proxy --labid some-labid --  python flask.py"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("Returns the command when it is provided in the correct format with sl config 2", func() {
			slConfig := &hooks.SealightsConfig{
				Token:     "",
				TokenFile: "some-token-file",
				BsId:      "",
				BsIdFile:  "some-bsid-file",
				Proxy:     "some-proxy",
				LabId:     "some-labid",
			}
			startCommand := "web: python flask.py"
			ModifiedCommand, err := sealights.GenerateStartUpCommand(startCommand, slConfig.GetStartFlags())
			Expect(ModifiedCommand).To(Equal("web: sl-python run --tokenfile some-token-file --buildsessionidfile some-bsid-file --proxy some-proxy --labid some-labid --  python flask.py"))
			Expect(err).NotTo(HaveOccurred())
		})
		It("Returns an error when provided the wrong format", func() {
			slConfig := hooks.NewSealightsConfig()
			startCommand := "python flask.py"
			_, err := sealights.GenerateStartUpCommand(startCommand, slConfig.GetStartFlags())
			Expect(err).To(MatchError("improper format found in Procfile"))
		})

	})

	Context("RewriteProcFile", func() {
		var (
			err         error
			tempProcDir string
			slConfig    *hooks.SealightsConfig
		)
		BeforeEach(func() {
			tempProcDir, err = os.MkdirTemp("", "Procfiles")
			Expect(err).NotTo(HaveOccurred())
			slConfig = hooks.NewSealightsConfig()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tempProcDir)).To(Succeed())
		})

		It("rewrites the procfile with sl-python", func() {
			Expect(os.WriteFile(filepath.Join(tempProcDir, "Procfile"), []byte("web: python app.py"), 0666)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
			err = sealights.RewriteProcFile(filepath.Join(tempProcDir, "Procfile"), slConfig.GetStartFlags())
			Expect(err).NotTo(HaveOccurred())
			startCommand, err := os.ReadFile(filepath.Join(tempProcDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(startCommand)).To(Equal("web: sl-python run --  python app.py"))
		})

		It("Errors when Procfile doesn't exist", func() {
			err := sealights.RewriteProcFile("/doesnt/exist", slConfig.GetStartFlags())
			Expect(err).To(MatchError("Error reading file /doesnt/exist: open /doesnt/exist: no such file or directory"))
		})

		It("Errors with Procfile with wrong format", func() {
			Expect(os.WriteFile(filepath.Join(tempProcDir, "WrongFormatProcFile"), []byte("python app.py"), 0666)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())

			err = sealights.RewriteProcFile(filepath.Join(tempProcDir, "WrongFormatProcFile"), slConfig.GetStartFlags())
			Expect(err).To(MatchError("improper format found in Procfile"))
		})
	})
	Context("RewriteRequirementsFile when requirements.txt is not packaged", func() {
		It("creates requirements.txt", func() {
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).ToNot(BeTrue())
			err := sealights.RewriteRequirementsFile(stager, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packagesList, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(packagesList)).To(Equal("sealights-python-agent"))
		})
	})

	Context("RewriteRequirementsFile when requirements.txt is packaged", func() {
		BeforeEach(func() {
			requirementsFilepath := filepath.Join(buildDir, "requirements.txt")
			Expect(libbuildpack.FileExists(requirementsFilepath)).ToNot(BeTrue())
			Expect(os.WriteFile(requirementsFilepath, []byte("Flask"), 0644)).To(Succeed())
		})
		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "requirements.txt"))).To(Succeed())
		})

		It("Rewrites requirements.txt", func() {
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			err := sealights.RewriteRequirementsFile(stager, "")
			Expect(err).NotTo(HaveOccurred())
			packages, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(string(packages)).To(Equal("Flask\nsealights-python-agent"))
		})
		It("Rewrites requirements.txt with sealights agent version", func() {
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			err := sealights.RewriteRequirementsFile(stager, "1.1.1")
			Expect(err).NotTo(HaveOccurred())
			packages, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(string(packages)).To(Equal("Flask\nsealights-python-agent==1.1.1"))
		})
	})
	Context("VCAP_SERVICES is not present", func() {
		BeforeEach(func() {
			Expect(os.Getenv("VCAP_SERVICES")).To(Equal(""))
			Expect(os.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("web: python app.py"), 0644)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "Procfile"))).To(Succeed())
		})

		It("BeforeCompile does not modify the existing Procfile", func() {
			err := sealights.BeforeCompile(stager)
			Expect(err).NotTo(HaveOccurred())
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeFalse())
			procCommand, err := os.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(procCommand)).To(Equal("web: python app.py"))
		})
	})
	Context("VCAP_SERVICES has no sealights", func() {
		BeforeEach(func() {
			Expect(os.Getenv("VCAP_SERVICES")).To(Equal(""))
			os.Setenv("VCAP_SERVICES", `{"service": [{"credentials": {"login": "name"}, "name": "443"}]}`)
			Expect(os.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("web: python app.py"), 0644)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.Remove(filepath.Join(buildDir, "Procfile"))).To(Succeed())
			os.Unsetenv("VCAP_SERVICES")
		})

		It("BeforeCompile does not modify the existing Procfile", func() {
			err := sealights.BeforeCompile(stager)
			Expect(err).NotTo(HaveOccurred())
			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeFalse())
			procCommand, err := os.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(procCommand)).To(Equal("web: python app.py"))
		})
	})

	Context("VCAP_SERVICES has sealights", func() {
		BeforeEach(func() {
			os.Setenv("VCAP_SERVICES", `{"sealights":[{"credentials":{"token":"","tokenFile":"token.txt"}}]}`)
			Expect(os.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("web: python app.py"), 0644)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_SERVICES")
			os.Unsetenv("SL_TOKEN")
		})
		It("BeforeCompile modify the existing Procfile", func() {
			err := sealights.BeforeCompile(stager)
			Expect(err).NotTo(HaveOccurred())

			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packages, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(packages)).To(Equal("sealights-python-agent"))

			procCommand, err := os.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(procCommand)).To(Equal("web: sl-python run --tokenfile token.txt --  python app.py"))
		})
		It("BeforeCompile modify the existing Procfile by override by env", func() {
			os.Setenv("SL_TOKEN", "token")
			err := sealights.BeforeCompile(stager)
			Expect(err).NotTo(HaveOccurred())

			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packages, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(packages)).To(Equal("sealights-python-agent"))

			procCommand, err := os.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(procCommand)).To(Equal("web: sl-python run --token token --  python app.py"))
		})

	})
	Context("VCAP_SERVICES has sealights with some prefix", func() {
		BeforeEach(func() {
			os.Setenv("VCAP_SERVICES", `{"some-sealights":[{"credentials":{"token":"","tokenFile":"token.txt"}}]}`)
			Expect(os.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("web: python app.py"), 0644)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_SERVICES")
		})
		It("BeforeCompile modify the existing Procfile", func() {
			err := sealights.BeforeCompile(stager)
			Expect(err).NotTo(HaveOccurred())

			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packages, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(packages)).To(Equal("sealights-python-agent"))

			procCommand, err := os.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(procCommand)).To(Equal("web: sl-python run --tokenfile token.txt --  python app.py"))
		})
	})
	Context("VCAP_SERVICES has user-provided sealights ", func() {
		BeforeEach(func() {
			os.Setenv("VCAP_SERVICES", `{"user-provided":[{"binding_guid":"af3fea1b-8beb-4397-968e-6440d2906551","binding_name":null,"credentials":{"token":"","tokenFile":"sltoken.txt"},"instance_guid":"d5c36671-dc09-4cd9-8697-70adc0c0f6e9","instance_name":"sealights","label":"user-provided","name":"sealights","syslog_drain_url":null,"tags":[],"volume_mounts":[]}]}`)
			Expect(os.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("web: python app.py"), 0644)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_SERVICES")
		})
		It("BeforeCompile modify the existing Procfile", func() {
			err := sealights.BeforeCompile(stager)
			Expect(err).NotTo(HaveOccurred())

			Expect(libbuildpack.FileExists(filepath.Join(buildDir, "requirements.txt"))).To(BeTrue())
			packages, err := os.ReadFile(filepath.Join(buildDir, "requirements.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(packages)).To(Equal("sealights-python-agent"))

			procCommand, err := os.ReadFile(filepath.Join(buildDir, "Procfile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(procCommand)).To(Equal("web: sl-python run --tokenfile sltoken.txt --  python app.py"))
		})
	})
	Context("VCAP_SERVICES has sealights and bad procfile", func() {
		BeforeEach(func() {
			os.Setenv("VCAP_SERVICES", `{"sealights":[{"credentials":{"token":"","tokenFile":"token.txt"}}]}`)
			Expect(os.WriteFile(filepath.Join(buildDir, "Procfile"), []byte("python app.py"), 0644)).To(Succeed())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			os.Unsetenv("VCAP_SERVICES")
		})
		It("BeforeCompile is not modify the existing Procfile -  bad format", func() {
			err := sealights.BeforeCompile(stager)
			Expect(err).To(MatchError("Failed to rewrite Procfile with Sealights: improper format found in Procfile"))

		})
	})
})
