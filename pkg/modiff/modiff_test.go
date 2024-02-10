package modiff_test

//nolint:revive // test file
import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/saschagrunert/go-modiff/pkg/modiff"
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
		from    = "v0.10.0"
		to      = "v0.11.0"
		badRepo = "github.com/saschagrunert/go-modiff-invalid"
	)

	BeforeEach(func() {
		logrus.SetLevel(logrus.PanicLevel)
	})

	It("should succeed", func() {
		// Given
		config := modiff.NewConfig(repo, from, to, false, 1)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(expected))
	})

	It("should succeed with links", func() {
		// Given
		config := modiff.NewConfig(repo, from, to, true, 1)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(expectedWithLinks))
	})

	It("should fail if context is nil", func() {
		// Given
		// When
		res, err := modiff.Run(nil)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'repository' not given", func() {
		// Given
		config := modiff.NewConfig("", from, to, true, 1)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if 'from' equals 'to'", func() {
		// Given
		config := modiff.NewConfig(repo, "", "", true, 1)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if repository is not clone-able", func() {
		// Given
		config := modiff.NewConfig("invalid", from, "", true, 1)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})

	It("should fail if the repository url is invalid", func() {
		// Given
		config := modiff.NewConfig(badRepo, from, to, true, 1)

		// When
		res, err := modiff.Run(config)

		// Then
		Expect(err).To(HaveOccurred())
		Expect(res).To(BeEmpty())
	})
})

func TestCheckURLValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		expected bool
		err      error
		client   *http.Client
	}{
		{
			name:     "Valid URL",
			url:      "https://github.com/hashicorp/consul/compare/api/v1.18.0...api/v1.20.0",
			expected: true,
			err:      nil,
		},
		{
			name:     "Invalid URL",
			url:      "https://github.com/hashicorp/consul/compare/v1.18.0...v1.20.0",
			expected: false,
			err:      nil,
		},
		{
			name:     "Request Sending Error",
			url:      "invalid-url",
			expected: false,
			err:      fmt.Errorf("error while sending request: "),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			if tt.client == nil {
				tt.client = &http.Client{}
			}
			valid, err := modiff.CheckURLValid(*tt.client, tt.url)
			g.Expect(valid).To(Equal(tt.expected))
			if tt.err != nil {
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring(tt.err.Error()))
			} else {
				g.Expect(err).ToNot(HaveOccurred())
			}
		})
	}
}
