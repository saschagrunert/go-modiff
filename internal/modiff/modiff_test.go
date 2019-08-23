package modiff_test

import (
	"flag"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/go-modiff/internal/modiff"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// The actual test suite
var _ = t.Describe("Run", func() {
	const expected = `# Dependencies

## Added
- github.com/saschagrunert/ccli: e981d95

## Changed
- github.com/fatih/color: v1.6.0 → v1.7.0
- github.com/mattn/go-colorable: v0.0.9 → v0.1.2
- github.com/mattn/go-isatty: v0.0.3 → v0.0.8

## Removed
_Nothing has changed._
`
	const (
		repo = "github.com/saschagrunert/go-modiff"
		from = "v0.1.0"
		to   = "v0.2.0"
	)
	var flagSet *flag.FlagSet

	BeforeEach(func() {
		logrus.SetLevel(logrus.PanicLevel)
		flagSet = flag.NewFlagSet("test", 0)
	})

	It("should succeed", func() {
		// Given
		flagSet.String(modiff.RepositoryArg, repo, "")
		flagSet.String(modiff.FromArg, from, "")
		flagSet.String(modiff.ToArg, to, "")
		context := cli.NewContext(nil, flagSet, nil)

		// When
		res, err := modiff.Run(context)

		// Then
		Expect(err).To(BeNil())
		Expect(res).To(Equal(expected))
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
		context := cli.NewContext(nil, flagSet, nil)

		// When
		res, err := modiff.Run(context)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'from' equals 'to'", func() {
		// Given
		flagSet.String(modiff.RepositoryArg, repo, "")
		context := cli.NewContext(nil, flagSet, nil)

		// When
		res, err := modiff.Run(context)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})

	It("should fail if repository is not clone-able", func() {
		// Given
		flagSet.String(modiff.RepositoryArg, "invalid", "")
		flagSet.String(modiff.FromArg, from, "")
		context := cli.NewContext(nil, flagSet, nil)

		// When
		res, err := modiff.Run(context)

		// Then
		Expect(err).NotTo(BeNil())
		Expect(res).To(BeEmpty())
	})
})
