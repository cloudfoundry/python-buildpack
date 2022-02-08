package requirements

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

const requirementsParserRegex = `(?m)^[\w\-\w\[\]]+`

type Reqs struct{}

func (m Reqs) FindAnyPackage(buildDir string, searchedPackages ...string) (bool, error) {
	reqPackages, err := parseRequirementsWithoutVersion(filepath.Join(buildDir, "requirements.txt"))
	if err != nil {
		return false, err
	}

	for _, searchedPackage := range searchedPackages {
		if containsPackage(reqPackages, searchedPackage) {
			return true, nil
		}
	}

	return false, nil
}

func (m Reqs) FindStalePackages(oldRequirementsPath, newRequirementsPath string, excludedPackages ...string) ([]string, error) {
	var stalePackages []string

	oldPkgs, err := parseRequirements(oldRequirementsPath)
	if err != nil {
		return nil, err
	}

	newPkgs, err := parseRequirements(newRequirementsPath)
	if err != nil {
		return nil, err
	}

	for _, oldPkg := range oldPkgs {
		if !containsPackage(newPkgs, oldPkg) && !packageIsExcluded(excludedPackages, oldPkg) {
			stalePackages = append(stalePackages, oldPkg)
		}
	}

	return stalePackages, nil
}

func containsPackage(packages []string, searchedPackage string) bool {
	for _, pkg := range packages {
		if pkg == searchedPackage {
			return true
		}
	}
	return false
}

func packageIsExcluded(excludedPackages []string, packageFullName string) bool {

	regex := regexp.MustCompile(requirementsParserRegex)

	packageWithoutVersion := regex.FindString(packageFullName)

	for _, excludedPackage := range excludedPackages {
		if packageWithoutVersion == excludedPackage {
			return true
		}
	}

	return false
}

func parseRequirements(requirementsPath string) ([]string, error) {
	content, err := ioutil.ReadFile(requirementsPath)
	if err != nil {
		return nil, err
	}

	requirements := strings.Split(string(content), "\n")

	for _, requirement := range requirements {
		if strings.HasPrefix(requirement, "-r ") {
			requirement = strings.TrimPrefix(requirement, "-r ")
			nestedReqs, err := parseRequirements(strings.ReplaceAll(requirementsPath, filepath.Base(requirementsPath), requirement))
			if err != nil {
				return nil, err
			}

			requirements = append(requirements, nestedReqs...)
		}
	}

	return requirements, nil
}

func parseRequirementsWithoutVersion(requirementsPath string) ([]string, error) {
	parsedRequirements, err := parseRequirements(requirementsPath)
	if err != nil {
		return nil, err
	}

	regex := regexp.MustCompile(`(?m)^[\w\-\w\[\]]+`)
	return regex.FindAllString(strings.Join(parsedRequirements, "\n"), -1), nil
}
