package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/libbuildpack"
	"os"
	"path/filepath"
	"strings"
)

type SealightsPlan struct {
	Credentials SealightsCredentials `json:"credentials"`
	Name        string               `json:"name,omitempty"`
}
type SealightsCredentials struct {
	Token     string `json:"token",omitempty`
	TokenFile string `json:"tokenFile",omitempty`
	BsId      string `json:"bsid",omitempty`
	BsIdFile  string `json:"bsidFile",omitempty`
	Proxy     string `json:"proxy",omitempty`
	LabId     string `json:"labid",omitempty`
	Version   string `json:"version",omitempty`
}
type SealightsConfig struct {
	Token     string
	TokenFile string
	BsId      string
	BsIdFile  string
	Proxy     string
	LabId     string
	Version   string
}

func NewSealightsConfig() *SealightsConfig {
	return &SealightsConfig{
		Token:     "",
		TokenFile: "",
		BsId:      "",
		BsIdFile:  "",
		Proxy:     "",
		LabId:     "",
		Version:   "",
	}
}

func (sc *SealightsConfig) getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
func (sc *SealightsConfig) parseSealightsPlan(plan SealightsPlan) *SealightsConfig {
	sc.Token = sc.getEnv("SL_TOKEN", plan.Credentials.Token)
	sc.TokenFile = sc.getEnv("SL_TOKEN_FILE", plan.Credentials.TokenFile)
	sc.BsId = sc.getEnv("SL_BUILD_SESSION_ID", plan.Credentials.BsId)
	sc.BsIdFile = sc.getEnv("SL_BUILD_SESSION_ID_FILE", plan.Credentials.BsIdFile)
	sc.Proxy = sc.getEnv("SL_PROXY", plan.Credentials.Proxy)
	sc.LabId = sc.getEnv("SL_LAB_ID", plan.Credentials.LabId)
	sc.Version = sc.getEnv("SL_VERSION", plan.Credentials.Version)
	return sc
}

func (sc *SealightsConfig) GetStartFlags() string {
	var flags []string
	if sc.Token != "" {
		flags = append(flags, fmt.Sprintf("--token %s", sc.Token))
	} else if sc.TokenFile != "" {
		flags = append(flags, fmt.Sprintf("--tokenfile %s", sc.TokenFile))
	}
	if sc.BsId != "" {
		flags = append(flags, fmt.Sprintf("--buildsessionid %s", sc.BsId))
	} else if sc.BsIdFile != "" {
		flags = append(flags, fmt.Sprintf("--buildsessionidfile %s", sc.BsIdFile))
	}
	if sc.Proxy != "" {
		flags = append(flags, fmt.Sprintf("--proxy %s", sc.Proxy))
	}
	if sc.LabId != "" {
		flags = append(flags, fmt.Sprintf("--labid %s", sc.LabId))
	}
	if len(flags) == 0 {
		return ""
	}
	return " " + strings.Join(flags, " ")
}

type SealightsHook struct {
	libbuildpack.DefaultHook
	Log *libbuildpack.Logger
}

func (sh SealightsHook) BeforeCompile(stager *libbuildpack.Stager) error {
	vcapServices := os.Getenv("VCAP_SERVICES")
	services := make(map[string][]SealightsPlan)

	err := json.Unmarshal([]byte(vcapServices), &services)
	if err != nil {
		sh.Log.Debug("Could not unmarshall VCAP_SERVICES JSON exiting: %v", err)
		return nil
	}

	sealightsServiceName, sealightsPlan := getSealightsServiceName(services)

	if sealightsServiceName == "" {
		sh.Log.Debug("No Sealights service found, exiting")
		return nil
	}
	sh.Log.BeginStep("Setting up Sealights hook")
	sealightsConfig := NewSealightsConfig().parseSealightsPlan(sealightsPlan)
	if err := sh.RewriteRequirementsFile(stager, sealightsConfig.Version); err != nil {
		sh.Log.Error("Could not write requirements file with Sealights package: %v", err)
		return err
	}

	if err := sh.RewriteProcFileWithSealights(stager, sealightsConfig.GetStartFlags()); err != nil {
		sh.Log.Error("Failed to rewrite Procfile with Sealights: %s", err.Error())
		return fmt.Errorf("Failed to rewrite Procfile with Sealights: %s", err.Error())
	}

	sh.Log.Info("Successfully set up Sealights hook")
	return nil
}
func (sh SealightsHook) RewriteProcFileWithSealights(stager *libbuildpack.Stager, cfgFlags string) error {
	sh.Log.BeginStep("Rewriting ProcFile to start with Sealights")

	file := filepath.Join(stager.BuildDir(), "Procfile")

	exists, err := libbuildpack.FileExists(file)
	if err != nil {
		return err
	}
	if exists {
		if err := sh.RewriteProcFile(file, cfgFlags); err != nil {
			return err
		}
		fileContents, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("Error reading file %s: %v", file, err)
		}
		sh.Log.Info(string(fileContents))
	} else {
		sh.Log.Info("Cannot find Procfile, skipping this step!")
	}
	return nil
}
func getSealightsServiceName(services map[string][]SealightsPlan) (string, SealightsPlan) {
	for serviceName, servicePlans := range services {
		if strings.Contains(serviceName, "sealights") {
			return serviceName, servicePlans[0]
		}
	}

	// checking if there is a user-provided service with name sealights
	userProvidedServices, keyExists := services["user-provided"]
	if !keyExists {
		return "", SealightsPlan{}
	}

	for _, plan := range userProvidedServices {
		if strings.Contains(plan.Name, "sealights") {
			return plan.Name, plan
		}
	}
	return "", SealightsPlan{}
}

func (sh SealightsHook) RewriteProcFile(procFilePath string, cfgFlags string) error {
	startCommand, err := os.ReadFile(procFilePath)
	if err != nil {
		return fmt.Errorf("Error reading file %s: %v", procFilePath, err)
	}
	newCommand, err := sh.GenerateStartUpCommand(string(startCommand), cfgFlags)
	if err != nil {
		return err
	}

	if err := os.WriteFile(procFilePath, []byte(newCommand), 0666); err != nil {
		return fmt.Errorf("Error writing file %s: %v", procFilePath, err)
	}
	return nil
}

func (sh SealightsHook) GenerateStartUpCommand(startCommand string, cfgFlags string) (string, error) {
	webCommands := strings.SplitN(startCommand, ":", 2)
	if len(webCommands) != 2 {
		return "", errors.New("improper format found in Procfile")
	}
	return fmt.Sprintf("web: sl-python run%s -- %s", cfgFlags, webCommands[1]), nil
}
func (sh SealightsHook) RewriteRequirementsFile(stager *libbuildpack.Stager, version string) error {
	sealightsPackage := "sealights-python-agent"
	if version != "" {
		sealightsPackage = fmt.Sprintf("sealights-python-agent==%s", version)
	}
	reqFile := filepath.Join(stager.BuildDir(), "requirements.txt")
	writeFlag := os.O_APPEND | os.O_WRONLY
	packageName := "\n" + sealightsPackage

	if exists, err := libbuildpack.FileExists(reqFile); err != nil {
		return err
	} else if !exists {
		sh.Log.Info("Requirements file not found creating one with sealights packages")
		writeFlag = os.O_CREATE | os.O_WRONLY
		packageName = sealightsPackage
	}
	f, err := os.OpenFile(reqFile, writeFlag, 0666)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	if _, err = f.WriteString(packageName); err != nil {
		return err
	}
	return nil
}

func init() {
	logger := libbuildpack.NewLogger(os.Stdout)
	libbuildpack.AddHook(&SealightsHook{
		Log: logger,
	})
}
