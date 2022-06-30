package modiff_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/saschagrunert/go-modiff/test/framework"
)

// TestModiff runs the created specs
func TestModiff(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunFrameworkSpecs(t, "modiff")
}

// nolint: gochecknoglobals
var t *TestFramework

var _ = BeforeSuite(func() {
	t = NewTestFramework(NilFunc, NilFunc)
	t.Setup()
})

var _ = AfterSuite(func() {
	t.Teardown()
})
