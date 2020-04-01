package modiff_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/go-modiff/internal/modiff"
	"github.com/sirupsen/logrus"
)

// The actual test suite
var _ = t.Describe("Run", func() {
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

	// nolint: lll
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
		repo = "github.com/saschagrunert/go-modiff"
		from = "v0.10.0"
		to   = "v0.11.0"
	)

	BeforeEach(func() {
		logrus.SetLevel(logrus.PanicLevel)
	})

	It("should succeed", func() {
		// Given
		config := modiff.NewConfig(repo, from, to, false)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).To(BeNil())
		Expect(res).To(Equal(expected))
	})

	It("should succeed with links", func() {
		// Given
		config := modiff.NewConfig(repo, from, to, true)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).To(BeNil())
		Expect(res).To(Equal(expectedWithLinks))
	})

	It("should fail if context is nil", func() {
		// Given
		// When
		res, err := modiff.Run(nil)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'repository' not given", func() {
		// Given
		config := modiff.NewConfig("", from, to, true)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'from' equals 'to'", func() {
		// Given
		config := modiff.NewConfig(repo, "", "", true)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})

	It("should fail if repository is not clone-able", func() {
		// Given
		config := modiff.NewConfig("invalid", from, "", true)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})
})
