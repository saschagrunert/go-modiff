// Package modiff provides functionality to diff Go module dependencies
// between two git revisions of a repository.
package modiff

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	gitHubPathSegments    = 3
	minModuleFields       = 2
	localRewriteFields    = 4
	fullRewriteFields     = 5
	minPseudoVersionParts = 3
	shortHashLength       = 7
	maxHeaderLevel        = 6
	refSplitParts         = 3
	goProxyDefault        = "https://proxy.golang.org"
	httpTimeoutSeconds    = 30

	// DefaultConcurrency is the default number of concurrent HTTP
	// requests when fetching module info from the proxy.
	DefaultConcurrency = 10

	// FormatMarkdown selects markdown output.
	FormatMarkdown = "markdown"

	// FormatJSON selects JSON output.
	FormatJSON = "json"

	// FilterAdded filters for added modules only.
	FilterAdded = "added"

	// FilterChanged filters for changed modules only.
	FilterChanged = "changed"

	// FilterRemoved filters for removed modules only.
	FilterRemoved = "removed"
)

var (
	errNilConfig       = errors.New("config is nil")
	errNoRepository    = errors.New("repository is required")
	errSameFromTo      = errors.New("no diff possible if `from` equals `to`")
	errProxyBadStatus  = errors.New("proxy returned unexpected status")
	errInvalidFormat   = errors.New("invalid format, must be markdown or json")
	errInvalidFilter   = errors.New("invalid filter, must be added, changed, or removed")
	errInvalidRepoPath = errors.New(
		"repository must be a valid module path (e.g., github.com/owner/repo)",
	)
)

type entry struct {
	beforeVersion string
	afterVersion  string
}

type modules map[string]entry

// ModuleChange represents a single module dependency change.
type ModuleChange struct {
	Name   string `json:"name"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Link   string `json:"link,omitempty"`
}

// DiffResult holds categorized module changes.
type DiffResult struct {
	Added   []ModuleChange `json:"added"`
	Changed []ModuleChange `json:"changed"`
	Removed []ModuleChange `json:"removed"`
}

// Config is the structure passed to `Run`.
type Config struct {
	repository  string
	from        string
	to          string
	link        bool
	headerLevel uint
	format      string
	filter      string
	concurrency uint
}

// NewConfig creates a new configuration.
func NewConfig(repository, from, to string, link bool, headerLevel uint) *Config {
	return &Config{
		repository:  repository,
		from:        from,
		to:          to,
		link:        link,
		headerLevel: headerLevel,
		format:      FormatMarkdown,
		filter:      "",
		concurrency: DefaultConcurrency,
	}
}

// WithFormat sets the output format (markdown or json).
func (c *Config) WithFormat(format string) *Config {
	c.format = format

	return c
}

// WithFilter sets the category filter (added, changed, or removed).
func (c *Config) WithFilter(filter string) *Config {
	c.filter = filter

	return c
}

// WithConcurrency sets the number of concurrent proxy requests.
func (c *Config) WithConcurrency(concurrency uint) *Config {
	c.concurrency = concurrency

	return c
}

// RunStructured starts go modiff and returns the structured diff result.
func RunStructured(ctx context.Context, config *Config) (DiffResult, error) {
	var empty DiffResult

	if config == nil {
		return empty, errNilConfig
	}

	if config.repository == "" {
		return empty, errNoRepository
	}

	if config.from == config.to {
		return empty, errSameFromTo
	}

	err := validateConfig(config)
	if err != nil {
		return empty, err
	}

	dir, err := os.MkdirTemp("", "go-modiff")
	if err != nil {
		return empty, fmt.Errorf("creating temp directory: %w", err)
	}

	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			slog.Error("Failed to remove temp dir", "error", err)
		}
	}()

	err = cloneRepos(ctx, dir, config)
	if err != nil {
		return empty, err
	}

	mods, err := getModules(ctx, filepath.Join(dir, "from"), filepath.Join(dir, "to"))
	if err != nil {
		return empty, err
	}

	result := diffModules(ctx, mods, config)
	applyFilter(&result, config.filter)

	return result, nil
}

// Run starts go modiff and returns the formatted result string.
func Run(ctx context.Context, config *Config) (string, error) {
	if config == nil {
		return "", errNilConfig
	}

	if config.format != FormatMarkdown && config.format != FormatJSON {
		return "", errInvalidFormat
	}

	result, err := RunStructured(ctx, config)
	if err != nil {
		return "", err
	}

	switch config.format {
	case FormatJSON:
		return formatJSON(result)
	default:
		return formatMarkdown(result, config.link, config.headerLevel, config.filter), nil
	}
}

func validateConfig(config *Config) error {
	if !strings.Contains(config.repository, ".") || !strings.Contains(config.repository, "/") {
		return errInvalidRepoPath
	}

	validFilters := map[string]bool{
		"": true, FilterAdded: true, FilterChanged: true, FilterRemoved: true,
	}

	if !validFilters[config.filter] {
		return errInvalidFilter
	}

	return nil
}
