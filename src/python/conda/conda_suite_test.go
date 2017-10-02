package conda_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConda(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conda Suite")
}
