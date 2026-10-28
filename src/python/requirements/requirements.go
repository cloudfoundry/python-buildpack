package requirements

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

const requirementsParserRegex = `(?m)^[\w\-\w\[\]]+`

var packageNameSeparatorRegex = regexp.MustCompile(`[-_.]+`)

type Reqs struct{}

func (m Reqs) FindAnyPackage(buildDir string, searchedPackages ...string) (bool, error) {
	requirementsPath := filepath.Join(buildDir, "requirements.txt")

	requirementsFileExists, err := libbuildpack.FileExists(requirementsPath)
	if err != nil {
		return false, err
	}

	if requirementsFileExists {

		reqPackages, err := parseRequirementsWithoutVersion(requirementsPath)
		if err != nil {
			return false, err
		}

		normalizedSearchedPackages := make([]string, len(searchedPackages))
		for i, searchedPackage := range searchedPackages {
			normalizedSearchedPackages[i] = normalizePackageName(searchedPackage)
		}

		for _, reqPackage := range reqPackages {
			normalizedReqPackage := normalizePackageName(reqPackage)
			for _, searchedPackage := range normalizedSearchedPackages {
				if normalizedReqPackage == searchedPackage {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func (m Reqs) FindStalePackages(oldRequirementsPath, newRequirementsPath string, excludedPackages ...string) ([]string, error) {
	var stalePackages []string

	oldRequirementsExists, err := libbuildpack.FileExists(oldRequirementsPath)
	if err != nil {
		return nil, err
	}

	newRequirementsExists, err := libbuildpack.FileExists(newRequirementsPath)
	if err != nil {
		return nil, err
	}

	if oldRequirementsExists && newRequirementsExists {

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

// normalizePackageName normalizes a package name as defined in PEP 503, since
// e.g. 'PyNaCl' may be written as 'pynacl' in generated requirements files.
func normalizePackageName(name string) string {
	return strings.ToLower(packageNameSeparatorRegex.ReplaceAllString(name, "-"))
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
	content, err := os.ReadFile(requirementsPath)
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
