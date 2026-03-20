package modiff_test

//nolint:revive // test file
import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/go-modiff/pkg/modiff"
	"github.com/sirupsen/logrus"
)

// The actual test suite.
var _ = testFramework.Describe("Run", func() {
	const expected = `# Dependencies

## Added
_Nothing has changed._

## Changed
- github.com/bombsimon/wsl: v1.2.5 → v1.2.1
- github.com/golangci/golangci-lint: v1.21.0 → v1.20.0
- github.com/golangci/lint-1: 297bf36 → fad67e0
- golang.org/x/tools: 0337d82 → 7c411de

## Removed
- github.com/gofrs/flock: 5135e61
`

	//nolint:lll // required formatting
	const expectedWithLinks = `# Dependencies

## Added
_Nothing has changed._

## Changed
- github.com/bombsimon/wsl: [v1.2.5 → v1.2.1](https://github.com/bombsimon/wsl/compare/v1.2.5...v1.2.1)
- github.com/golangci/golangci-lint: [v1.21.0 → v1.20.0](https://github.com/golangci/golangci-lint/compare/v1.21.0...v1.20.0)
- github.com/golangci/lint-1: [297bf36 → fad67e0](https://github.com/golangci/lint-1/compare/297bf36...fad67e0)
- golang.org/x/tools: 0337d82 → 7c411de

## Removed
- github.com/gofrs/flock: [5135e61](https://github.com/gofrs/flock/tree/5135e61)
`

	const (
		repo    = "github.com/saschagrunert/go-modiff"
		fromRev = "v0.10.0"
		toRev   = "v0.11.0"
		badRepo = "github.com/saschagrunert/go-modiff-invalid"
	)

	BeforeEach(func() {
		logrus.SetLevel(logrus.PanicLevel)
	})

	It("should succeed", func() {
		// Given
		config := modiff.NewConfig(repo, fromRev, toRev, false, 1)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(expected))
	})

	It("should succeed with links", func() {
		// Given
		config := modiff.NewConfig(repo, fromRev, toRev, true, 1)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(expectedWithLinks))
	})

	It("should fail if context is nil", func() {
		// Given
		// When
		res, err := modiff.Run(context.Background(), nil)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'repository' not given", func() {
		// Given
		config := modiff.NewConfig("", fromRev, toRev, true, 1)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'from' equals 'to'", func() {
		// Given
		config := modiff.NewConfig(repo, "", "", true, 1)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if repository is not clone-able", func() {
		// Given
		config := modiff.NewConfig("invalid", fromRev, "", true, 1)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if the repository url is invalid", func() {
		// Given
		config := modiff.NewConfig(badRepo, fromRev, toRev, true, 1)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})
})

var errSending = errors.New("error while sending request: ")

func TestCheckURLValid(test *testing.T) {
	test.Parallel()

	test.Run("Valid URL", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{} //nolint:exhaustruct // zero value is fine
		valid, err := modiff.CheckURLValid(context.Background(), *client, server.URL)
		gomega.Expect(err).ToNot(HaveOccurred())
		gomega.Expect(valid).To(BeTrue())
	})

	test.Run("Invalid URL (404)", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := &http.Client{} //nolint:exhaustruct // zero value is fine
		valid, err := modiff.CheckURLValid(context.Background(), *client, server.URL)
		gomega.Expect(err).ToNot(HaveOccurred())
		gomega.Expect(valid).To(BeFalse())
	})

	test.Run("Request Sending Error", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		client := &http.Client{} //nolint:exhaustruct // zero value is fine
		valid, err := modiff.CheckURLValid(context.Background(), *client, "invalid-url")
		gomega.Expect(err).To(HaveOccurred())
		gomega.Expect(err.Error()).To(ContainSubstring(errSending.Error()))
		gomega.Expect(valid).To(BeFalse())
	})
}
