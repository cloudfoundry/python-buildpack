package integration_test

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry/libbuildpack/cutlass"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type haveLoggedMatcher struct {
	expected string
}

func HaveLogged(expected string) types.GomegaMatcher {
	return &haveLoggedMatcher{expected: expected}
}

func (matcher *haveLoggedMatcher) Match(actual interface{}) (success bool, err error) {
	app, ok := actual.(*cutlass.App)
	if !ok {
		return false, fmt.Errorf("HaveLogged matcher requires a cutlass.App.  Got:\n%s", format.Object(actual, 1))
	}

	return strings.Contains(app.Stdout.String(), matcher.expected), nil
}

func (matcher *haveLoggedMatcher) FailureMessage(actual interface{}) (message string) {
	app := actual.(*cutlass.App)
	return format.Message(app.Stdout.String(), "to contain substring", matcher.expected)
}

func (matcher *haveLoggedMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	app := actual.(*cutlass.App)
	return format.Message(app.Stdout.String(), "not to contain substring", matcher.expected)
}
