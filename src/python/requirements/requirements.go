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

// packageNameRegex matches the package name of a requirement, including dots
// (e.g. 'zope.interface') and extras (e.g. 'django[argon2]').
var packageNameRegex = regexp.MustCompile(`(?m)^[\w\-.]+(?:\[[^\]]*\])?`)

var packageExtrasRegex = regexp.MustCompile(`\[(.*)\]`)

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

		for _, reqPackage := range reqPackages {
			for _, searchedPackage := range searchedPackages {
				if packageMatches(reqPackage, searchedPackage) {
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

// packageMatches returns whether a requirement like 'Django[argon2]' matches
// a searched package. The names are compared normalized, and the requirement
// must have all extras of the searched package, so that 'django' matches
// 'django[argon2]', but 'django[argon2]' does not match 'django'.
func packageMatches(reqPackage, searchedPackage string) bool {
	reqName, reqExtras := splitPackageExtras(reqPackage)
	searchedName, searchedExtras := splitPackageExtras(searchedPackage)
	if reqName != searchedName {
		return false
	}

	for _, searchedExtra := range searchedExtras {
		if !containsPackage(reqExtras, searchedExtra) {
			return false
		}
	}
	return true
}

// splitPackageExtras returns the normalized name and extras of a package.
func splitPackageExtras(pkg string) (string, []string) {
	var extras []string
	if match := packageExtrasRegex.FindStringSubmatch(pkg); match != nil {
		for _, extra := range strings.Split(match[1], ",") {
			extras = append(extras, normalizePackageName(strings.TrimSpace(extra)))
		}
	}
	return normalizePackageName(packageExtrasRegex.ReplaceAllString(pkg, "")), extras
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

	return packageNameRegex.FindAllString(strings.Join(parsedRequirements, "\n"), -1), nil
}
