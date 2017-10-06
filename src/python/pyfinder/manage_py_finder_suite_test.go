package pyfinder_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestManagePyFinder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ManagePyFinder Suite")
}
