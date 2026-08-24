package modiff_test

import (
	"context"
	"log/slog"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/saschagrunert/go-modiff/pkg/modiff"
)

// The actual test suite.
var _ = Describe("Run", func() {
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
		slog.SetDefault(slog.New(slog.DiscardHandler))
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

	It("should fail if repository path is invalid", func() {
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

func TestRefName(test *testing.T) {
	test.Parallel()

	test.Run("Full ref path", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar",
			"abc123",
			"refs/tags/v1.0.0",
		)

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

		gomega.Expect(modiff.CommitURLForTest(info)).
			To(Equal("https://github.com/foo/bar/commit/abc123"))
	})

	test.Run("Googlesource", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		info := modiff.NewGoModInfoForTest("https://go.googlesource.com/tools", "abc123", "")

		gomega.Expect(modiff.CommitURLForTest(info)).
			To(Equal("https://go.googlesource.com/tools/+/abc123"))
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
		info := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar",
			"abc123",
			"refs/tags/v1.0.0",
		)
		other := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar",
			"def456",
			"refs/tags/v2.0.0",
		)

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

	test.Run("Pipe separated", func(subTest *testing.T) {
		subTest.Setenv("GOPROXY", "https://first.example.com|https://second.example.com")

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

		name, version, ok := modiff.ParseModuleLineForTest(
			"github.com/old v1.0.0 => github.com/new v2.0.0",
		)

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

func TestApplyFilter(test *testing.T) {
	test.Parallel()

	newResult := func() modiff.DiffResult {
		return modiff.DiffResult{
			Added:   []modiff.ModuleChange{{Name: "a", Before: "", After: "v1", Link: ""}},
			Changed: []modiff.ModuleChange{{Name: "b", Before: "v1", After: "v2", Link: ""}},
			Removed: []modiff.ModuleChange{{Name: "c", Before: "v1", After: "", Link: ""}},
		}
	}

	test.Run("Filter added", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		result := newResult()
		modiff.ApplyFilterForTest(&result, modiff.FilterAdded)

		gomega.Expect(result.Added).To(HaveLen(1))
		gomega.Expect(result.Changed).To(BeEmpty())
		gomega.Expect(result.Removed).To(BeEmpty())
	})

	test.Run("Filter changed", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		result := newResult()
		modiff.ApplyFilterForTest(&result, modiff.FilterChanged)

		gomega.Expect(result.Added).To(BeEmpty())
		gomega.Expect(result.Changed).To(HaveLen(1))
		gomega.Expect(result.Removed).To(BeEmpty())
	})

	test.Run("Filter removed", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		result := newResult()
		modiff.ApplyFilterForTest(&result, modiff.FilterRemoved)

		gomega.Expect(result.Added).To(BeEmpty())
		gomega.Expect(result.Changed).To(BeEmpty())
		gomega.Expect(result.Removed).To(HaveLen(1))
	})

	test.Run("No filter", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		result := newResult()
		modiff.ApplyFilterForTest(&result, "")

		gomega.Expect(result.Added).To(HaveLen(1))
		gomega.Expect(result.Changed).To(HaveLen(1))
		gomega.Expect(result.Removed).To(HaveLen(1))
	})
}

func TestClassifyModule(test *testing.T) {
	test.Parallel()

	test.Run("Unchanged module", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("v1.0.0", "v1.0.0")

		category, change := modiff.ClassifyModuleForTest(mod, "github.com/foo/bar", nil, nil, false)

		gomega.Expect(category).To(BeEmpty())
		gomega.Expect(change.Name).To(BeEmpty())
	})
}

func TestGenerateSingleLink(test *testing.T) {
	test.Parallel()

	test.Run("Non-GitHub module without info", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		link := modiff.GenerateSingleLinkForTest("v1.0.0", nil, "gitlab.com/foo/bar")

		gomega.Expect(link).To(BeEmpty())
	})

	test.Run("GitHub module without info", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		link := modiff.GenerateSingleLinkForTest("v1.0.0", nil, "github.com/foo/bar")

		gomega.Expect(link).To(Equal("https://github.com/foo/bar/tree/v1.0.0"))
	})
}

func TestGitHubBaseURL(test *testing.T) {
	test.Parallel()

	test.Run("Standard three-segment path", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(modiff.GitHubBaseURLForTest("github.com/foo/bar")).To(
			Equal("https://github.com/foo/bar"),
		)
	})

	test.Run("Deep module path", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(modiff.GitHubBaseURLForTest("github.com/foo/bar/v2")).To(
			Equal("https://github.com/foo/bar"),
		)
	})

	test.Run("Short path fallback", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		gomega.Expect(modiff.GitHubBaseURLForTest("github.com/foo")).To(
			Equal("https://github.com/foo"),
		)
	})
}

func TestWithConcurrency(test *testing.T) {
	test.Parallel()

	gomega := NewGomegaWithT(test)
	config := modiff.NewConfig("github.com/foo/bar", "v1", "v2", true, 1).
		WithConcurrency(5)

	gomega.Expect(config.ConcurrencyForTest()).To(Equal(uint(5)))
}

func TestRunStructured(test *testing.T) {
	test.Parallel()

	test.Run("Nil config", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		result, err := modiff.RunStructured(context.Background(), nil)

		gomega.Expect(err).To(HaveOccurred())
		gomega.Expect(err.Error()).To(ContainSubstring("config is nil"))
		gomega.Expect(result.Added).To(BeNil())
	})

	test.Run("Invalid repo path", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		config := modiff.NewConfig("invalid", "v1", "v2", false, 1)
		result, err := modiff.RunStructured(context.Background(), config)

		gomega.Expect(err).To(HaveOccurred())
		gomega.Expect(err.Error()).To(ContainSubstring("valid module path"))
		gomega.Expect(result.Added).To(BeNil())
	})

	test.Run("Empty repository", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		config := modiff.NewConfig("", "v1", "v2", false, 1)
		result, err := modiff.RunStructured(context.Background(), config)

		gomega.Expect(err).To(HaveOccurred())
		gomega.Expect(err.Error()).To(ContainSubstring("repository is required"))
		gomega.Expect(result.Added).To(BeNil())
	})

	test.Run("From equals to", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		config := modiff.NewConfig("github.com/foo/bar", "v1", "v1", false, 1)
		result, err := modiff.RunStructured(context.Background(), config)

		gomega.Expect(err).To(HaveOccurred())
		gomega.Expect(err.Error()).To(ContainSubstring("from"))
		gomega.Expect(result.Added).To(BeNil())
	})
}

func TestFormatModuleMarkdown(test *testing.T) {
	test.Parallel()

	test.Run("Added without link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{Name: "github.com/foo/bar", After: "v1.0.0"}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterAdded, false)

		gomega.Expect(result).To(Equal("- github.com/foo/bar: v1.0.0"))
	})

	test.Run("Added with link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{
			Name:  "github.com/foo/bar",
			After: "v1.0.0",
			Link:  "https://github.com/foo/bar/commit/abc",
		}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterAdded, true)

		gomega.Expect(result).To(Equal(
			"- github.com/foo/bar: [v1.0.0](https://github.com/foo/bar/commit/abc)",
		))
	})

	test.Run("Removed without link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{Name: "github.com/foo/bar", Before: "v1.0.0"}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterRemoved, false)

		gomega.Expect(result).To(Equal("- github.com/foo/bar: v1.0.0"))
	})

	test.Run("Removed with link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{
			Name:   "github.com/foo/bar",
			Before: "v1.0.0",
			Link:   "https://github.com/foo/bar/commit/abc",
		}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterRemoved, true)

		gomega.Expect(result).To(Equal(
			"- github.com/foo/bar: [v1.0.0](https://github.com/foo/bar/commit/abc)",
		))
	})

	test.Run("Changed without link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{
			Name:   "github.com/foo/bar",
			Before: "v1.0.0",
			After:  "v2.0.0",
		}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterChanged, false)

		gomega.Expect(result).To(Equal("- github.com/foo/bar: v1.0.0 → v2.0.0"))
	})

	test.Run("Changed with link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{
			Name:   "github.com/foo/bar",
			Before: "v1.0.0",
			After:  "v2.0.0",
			Link:   "https://github.com/foo/bar/compare/v1.0.0...v2.0.0",
		}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterChanged, true)

		gomega.Expect(result).To(Equal(
			"- github.com/foo/bar: [v1.0.0 → v2.0.0](https://github.com/foo/bar/compare/v1.0.0...v2.0.0)",
		))
	})

	test.Run("Added with link but empty link string", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		change := modiff.ModuleChange{Name: "github.com/foo/bar", After: "v1.0.0"}

		result := modiff.FormatModuleMarkdownForTest(change, modiff.FilterAdded, true)

		gomega.Expect(result).To(Equal("- github.com/foo/bar: v1.0.0"))
	})
}

func TestBuildDiffResult(test *testing.T) {
	test.Parallel()

	test.Run("Sorts and categorizes", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		categories := []string{
			modiff.FilterAdded, modiff.FilterAdded,
			modiff.FilterChanged,
			modiff.FilterRemoved,
		}
		changes := []modiff.ModuleChange{
			{Name: "z-mod", After: "v1"},
			{Name: "a-mod", After: "v2"},
			{Name: "m-mod", Before: "v1", After: "v2"},
			{Name: "r-mod", Before: "v1"},
		}

		result := modiff.BuildDiffResultForTest(categories, changes)

		gomega.Expect(result.Added).To(HaveLen(2))
		gomega.Expect(result.Added[0].Name).To(Equal("a-mod"))
		gomega.Expect(result.Added[1].Name).To(Equal("z-mod"))
		gomega.Expect(result.Changed).To(HaveLen(1))
		gomega.Expect(result.Removed).To(HaveLen(1))
	})

	test.Run("Empty input", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)

		result := modiff.BuildDiffResultForTest([]string{}, []modiff.ModuleChange{})

		gomega.Expect(result.Added).To(BeEmpty())
		gomega.Expect(result.Changed).To(BeEmpty())
		gomega.Expect(result.Removed).To(BeEmpty())
	})
}

func TestClassifyModuleCategories(test *testing.T) {
	test.Parallel()

	test.Run("Added module", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("", "v1.0.0")

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", nil, nil, false,
		)

		gomega.Expect(category).To(Equal(modiff.FilterAdded))
		gomega.Expect(change.Name).To(Equal("github.com/foo/bar"))
		gomega.Expect(change.After).To(Equal("v1.0.0"))
	})

	test.Run("Removed module", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("v1.0.0", "")

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", nil, nil, false,
		)

		gomega.Expect(category).To(Equal(modiff.FilterRemoved))
		gomega.Expect(change.Name).To(Equal("github.com/foo/bar"))
		gomega.Expect(change.Before).To(Equal("v1.0.0"))
	})

	test.Run("Changed module", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("v1.0.0", "v2.0.0")

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", nil, nil, false,
		)

		gomega.Expect(category).To(Equal(modiff.FilterChanged))
		gomega.Expect(change.Name).To(Equal("github.com/foo/bar"))
		gomega.Expect(change.Before).To(Equal("v1.0.0"))
		gomega.Expect(change.After).To(Equal("v2.0.0"))
	})

	test.Run("Added with link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("", "v1.0.0")
		info := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar", "abc123", "refs/tags/v1.0.0",
		)

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", nil, info, true,
		)

		gomega.Expect(category).To(Equal(modiff.FilterAdded))
		gomega.Expect(change.Link).ToNot(BeEmpty())
	})

	test.Run("Removed with link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("v1.0.0", "")
		info := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar", "abc123", "refs/tags/v1.0.0",
		)

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", info, nil, true,
		)

		gomega.Expect(category).To(Equal(modiff.FilterRemoved))
		gomega.Expect(change.Link).ToNot(BeEmpty())
	})

	test.Run("Changed with links", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("v1.0.0", "v2.0.0")
		before := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar", "abc123", "refs/tags/v1.0.0",
		)
		after := modiff.NewGoModInfoForTest(
			"https://github.com/foo/bar", "def456", "refs/tags/v2.0.0",
		)

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", before, after, true,
		)

		gomega.Expect(category).To(Equal(modiff.FilterChanged))
		gomega.Expect(change.Link).To(
			Equal("https://github.com/foo/bar/compare/v1.0.0...v2.0.0"),
		)
	})

	test.Run("Changed GitHub fallback link", func(subTest *testing.T) {
		subTest.Parallel()

		gomega := NewGomegaWithT(subTest)
		mod := modiff.NewEntryForTest("v1.0.0", "v2.0.0")

		category, change := modiff.ClassifyModuleForTest(
			mod, "github.com/foo/bar", nil, nil, true,
		)

		gomega.Expect(category).To(Equal(modiff.FilterChanged))
		gomega.Expect(change.Link).To(
			Equal("https://github.com/foo/bar/compare/v1.0.0...v2.0.0"),
		)
	})
}
