package supply_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/ini.v1"

	"testing"
)

func TestSupply(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Supply Suite")
}

// This will only parse [easy_install] section and return map
func ParsePydistutils(contents string) (map[string][]string, error) {
	contentBytes := []byte(contents)
	easyInstall := map[string][]string{}
	loadOpts := ini.LoadOptions{
		AllowPythonMultilineValues: true,
	}

	iniContent, err := ini.LoadSources(loadOpts, contentBytes)
	if err != nil {
		return map[string][]string{}, err
	}

	for _, key := range iniContent.Section("easy_install").KeyStrings() {
		easyInstall[key] = iniContent.Section("easy_install").Key(key).Strings("\n")
	}

	return easyInstall, nil
}
