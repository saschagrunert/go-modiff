package modiff_test

//nolint:revive // test file
import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/saschagrunert/go-modiff/test/framework"
)

// TestModiff runs the created specs.
func TestModiff(test *testing.T) {
	test.Parallel()
	RegisterFailHandler(Fail)
	RunFrameworkSpecs(test, "modiff")
}

//nolint:gochecknoglobals // test framework should be global
var testFramework *TestFramework

var _ = BeforeSuite(func() {
	testFramework = NewTestFramework(NilFunc, NilFunc)
	testFramework.Setup()
})

var _ = AfterSuite(func() {
	testFramework.Teardown()
})
