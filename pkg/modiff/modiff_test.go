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

	It("should succeed with JSON format", func() {
		// Given
		config := modiff.NewConfig(repo, fromRev, toRev, false, 1).
			WithFormat(modiff.FormatJSON)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(ContainSubstring(`"changed"`))
		Expect(res).To(ContainSubstring(`"github.com/bombsimon/wsl"`))
	})

	It("should succeed with filter", func() {
		// Given
		config := modiff.NewConfig(repo, fromRev, toRev, false, 1).
			WithFilter(modiff.FilterRemoved)

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(ContainSubstring("Removed"))
		Expect(res).ToNot(ContainSubstring("## Added"))
		Expect(res).ToNot(ContainSubstring("## Changed"))
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

	It("should fail with invalid format", func() {
		// Given
		config := modiff.NewConfig(repo, fromRev, toRev, false, 1).
			WithFormat("xml")

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid format"))
		Expect(res).To(BeEmpty())
	})

	It("should fail with invalid filter", func() {
		// Given
		config := modiff.NewConfig(repo, fromRev, toRev, false, 1).
			WithFilter("bogus")

		// When
		res, err := modiff.Run(context.Background(), config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid filter"))
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

func TestRefName(test *testing.T) {
	test.Parallel()

	test.Run("Full ref path", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "refs/tags/v1.0.0")

		gomega.Expect(modiff.RefNameForTest(info)).To(Equal("v1.0.0"))
	})

	test.Run("Hash only", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "")

		gomega.Expect(modiff.RefNameForTest(info)).To(Equal("abc123"))
	})

	test.Run("Simple ref without slashes", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "v1.0.0")

		gomega.Expect(modiff.RefNameForTest(info)).To(Equal("v1.0.0"))
	})

	test.Run("Ref with two parts", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "tags/v1.0.0")

		gomega.Expect(modiff.RefNameForTest(info)).To(Equal("tags/v1.0.0"))
	})
}

func TestCommitURL(test *testing.T) {
	test.Parallel()

	test.Run("GitHub", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "")

		gomega.Expect(modiff.CommitURLForTest(info)).To(Equal("https://github.com/foo/bar/commit/abc123"))
	})

	test.Run("Googlesource", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://go.googlesource.com/tools", "abc123", "")

		gomega.Expect(modiff.CommitURLForTest(info)).To(Equal("https://go.googlesource.com/tools/+/abc123"))
	})

	test.Run("Unknown host", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://gitlab.com/foo/bar", "abc123", "")

		gomega.Expect(modiff.CommitURLForTest(info)).To(BeEmpty())
	})
}

func TestCompareURL(test *testing.T) {
	test.Parallel()

	test.Run("GitHub with refs", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "refs/tags/v1.0.0")
		other := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "def456", "refs/tags/v2.0.0")

		gomega.Expect(modiff.CompareURLForTest(info, other)).To(
			Equal("https://github.com/foo/bar/compare/v1.0.0...v2.0.0"),
		)
	})

	test.Run("GitHub with hashes", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "abc123", "")
		other := modiff.NewGoModInfoForTest("https://github.com/foo/bar", "def456", "")

		gomega.Expect(modiff.CompareURLForTest(info, other)).To(
			Equal("https://github.com/foo/bar/compare/abc123...def456"),
		)
	})

	test.Run("Googlesource", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://go.googlesource.com/tools", "abc123", "")
		other := modiff.NewGoModInfoForTest("https://go.googlesource.com/tools", "def456", "")

		gomega.Expect(modiff.CompareURLForTest(info, other)).To(
			Equal("https://go.googlesource.com/tools/+/abc123^1..def456/"),
		)
	})

	test.Run("Unknown host", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://gitlab.com/foo/bar", "abc123", "")
		other := modiff.NewGoModInfoForTest("https://gitlab.com/foo/bar", "def456", "")

		gomega.Expect(modiff.CompareURLForTest(info, other)).To(BeEmpty())
	})
}

//nolint:paralleltest // cannot parallelize environment variable tests
func TestGoProxyURL(test *testing.T) {
	test.Run("Default", func(subTest *testing.T) {
		gomega := NewGomegaWithT(subTest)

		result := modiff.GoProxyURLForTest()

		gomega.Expect(result).To(Equal("https://proxy.golang.org"))
	})

	test.Run("Custom proxy", func(subTest *testing.T) {
		subTest.Setenv("GOPROXY", "https://custom.proxy.example.com")

		gomega := NewGomegaWithT(subTest)

		result := modiff.GoProxyURLForTest()

		gomega.Expect(result).To(Equal("https://custom.proxy.example.com"))
	})

	test.Run("Direct fallback", func(subTest *testing.T) {
		subTest.Setenv("GOPROXY", "direct")

		gomega := NewGomegaWithT(subTest)

		result := modiff.GoProxyURLForTest()

		gomega.Expect(result).To(Equal("https://proxy.golang.org"))
	})

	test.Run("Off fallback", func(subTest *testing.T) {
		subTest.Setenv("GOPROXY", "off")

		gomega := NewGomegaWithT(subTest)

		result := modiff.GoProxyURLForTest()

		gomega.Expect(result).To(Equal("https://proxy.golang.org"))
	})

	test.Run("Comma separated", func(subTest *testing.T) {
		subTest.Setenv("GOPROXY", "https://first.example.com,https://second.example.com")

		gomega := NewGomegaWithT(subTest)

		result := modiff.GoProxyURLForTest()

		gomega.Expect(result).To(Equal("https://first.example.com"))
	})

	test.Run("Empty value", func(subTest *testing.T) {
		subTest.Setenv("GOPROXY", "")

		gomega := NewGomegaWithT(subTest)

		result := modiff.GoProxyURLForTest()

		gomega.Expect(result).To(Equal("https://proxy.golang.org"))
	})
}

func TestParseModuleLine(test *testing.T) {
	test.Parallel()

	test.Run("Simple module", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		name, version, ok := modiff.ParseModuleLineForTest("github.com/foo/bar v1.0.0")

		gomega.Expect(ok).To(BeTrue())
		gomega.Expect(name).To(Equal("github.com/foo/bar"))
		gomega.Expect(version).To(Equal("v1.0.0"))
	})

	test.Run("Too few fields", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		_, _, ok := modiff.ParseModuleLineForTest("github.com/foo/bar")

		gomega.Expect(ok).To(BeFalse())
	})

	test.Run("Local rewrite skipped", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		_, _, ok := modiff.ParseModuleLineForTest("github.com/foo/bar v1.0.0 => ../local")

		gomega.Expect(ok).To(BeFalse())
	})

	test.Run("Full rewrite", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		name, version, ok := modiff.ParseModuleLineForTest("github.com/old v1.0.0 => github.com/new v2.0.0")

		gomega.Expect(ok).To(BeTrue())
		gomega.Expect(name).To(Equal("github.com/new"))
		gomega.Expect(version).To(Equal("v2.0.0"))
	})

	test.Run("Empty line", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		_, _, ok := modiff.ParseModuleLineForTest("")

		gomega.Expect(ok).To(BeFalse())
	})
}

func TestPrettifyVersion(test *testing.T) {
	test.Parallel()

	test.Run("Semantic version", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(modiff.PrettifyVersionForTest("v1.2.3")).To(Equal("v1.2.3"))
	})

	test.Run("Pseudo version", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(
			modiff.PrettifyVersionForTest("v0.0.0-20210101120000-abcdef1234567"),
		).To(Equal("abcdef1"))
	})

	test.Run("Incompatible suffix", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(modiff.PrettifyVersionForTest("v2.0.0+incompatible")).To(Equal("v2.0.0"))
	})

	test.Run("Short hash", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(modiff.PrettifyVersionForTest("v0.0.0-20210101-abc")).To(Equal("abc"))
	})
}
