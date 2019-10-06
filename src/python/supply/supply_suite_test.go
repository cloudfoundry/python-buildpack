package supply_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bufio"
	"regexp"
	"strings"
	"testing"
)

func TestSupply(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Supply Suite")
}

// This will only parse [easy_install] section and return map
func ParsePydistutils(contents string) map[string][]string {
	configMap := make(map[string][]string)
	scanner := bufio.NewScanner(strings.NewReader(contents))
	var isEasyInstall = false
	var currentKey = ""

	// regular expression for `key = value`.
	kv := regexp.MustCompile(`^([^=\n]+)=([^\n]*)`)
	for scanner.Scan() {
		line := scanner.Text()
		if line[0] == '[' {
			if strings.Compare("[easy_install]", line) == 0 {
				isEasyInstall = true
			} else {
				isEasyInstall = false
			}
		} else if isEasyInstall {
			m := kv.FindStringSubmatch(line)
			if len(m) > 0 {
				currentKey = strings.Trim(m[len(m)-2], " ")
				value := strings.Trim(m[len(m)-1], "\r\n\t ")
				configMap[currentKey] = []string{value}
			} else {
				value := strings.Trim(line, "\r\n\t ")
				configMap[currentKey] = append(configMap[currentKey], value)
			}
		}
	}
	return configMap
}
