package modiff_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestModiff(test *testing.T) {
	test.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(test, "modiff")
}
