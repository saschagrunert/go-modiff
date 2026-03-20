// Package modiff provides functionality to diff Go module dependencies
// between two git revisions of a repository.
package modiff

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// gitHubPathSegments is the number of path segments in a standard
	// GitHub module path (e.g. github.com/owner/repo).
	gitHubPathSegments = 3

	// minModuleFields is the minimum number of fields expected when
	// parsing a go list module line (name + version).
	minModuleFields = 2

	// localRewriteFields is the number of fields for a local rewrite
	// without a version (e.g. "mod => ../local").
	localRewriteFields = 4

	// fullRewriteFields is the number of fields for a rewrite with
	// a version (e.g. "old v1 => new v2").
	fullRewriteFields = 5

	// minPseudoVersionParts is the minimum number of dash-separated
	// parts in a pseudo-version string.
	minPseudoVersionParts = 3

	// shortHashLength is the length used to truncate pseudo-version
	// commit hashes for display.
	shortHashLength = 7

	// maxHeaderLevel is the maximum markdown header level (h6).
	maxHeaderLevel = 6

	// refSplitParts is the number of parts when splitting a git ref
	// path (e.g. refs/tags/v1.0.0 splits into 3 parts).
	refSplitParts = 3

	// proxySplitParts is the number of parts when splitting GOPROXY
	// env value at the first comma.
	proxySplitParts = 2

	// goProxyDefault is the default Go module proxy URL.
	goProxyDefault = "https://proxy.golang.org"

	// DefaultConcurrency is the default number of concurrent HTTP
	// requests when fetching module info from the proxy.
	DefaultConcurrency = 10

	// httpTimeoutSeconds is the timeout in seconds for HTTP requests
	// to the Go module proxy.
	httpTimeoutSeconds = 30

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
	errNilConfig      = errors.New("config is nil")
	errNoRepository   = errors.New("repository is required")
	errSameFromTo     = errors.New("no diff possible if `from` equals `to`")
	errProxyBadStatus = errors.New("proxy returned unexpected status")
	errInvalidFormat  = errors.New("invalid format, must be markdown or json")
	errInvalidFilter  = errors.New("invalid filter, must be added, changed, or removed")
)

// goModOrigin holds VCS origin information from the Go module proxy.
type goModOrigin struct {
	VCS  string `json:"VCS"`  //nolint:tagliatelle // matches proxy response format
	URL  string `json:"URL"`  //nolint:tagliatelle // matches proxy response format
	Hash string `json:"Hash"` //nolint:tagliatelle // matches proxy response format
	Ref  string `json:"Ref"`  //nolint:tagliatelle // matches proxy response format
}

// goModInfo holds module metadata from the Go module proxy.
type goModInfo struct {
	Version string      `json:"Version"` //nolint:tagliatelle // matches proxy response format
	Origin  goModOrigin `json:"Origin"`  //nolint:tagliatelle // matches proxy response format
}

type entry struct {
	beforeVersion string
	afterVersion  string
}

type modules = map[string]entry

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

// Run starts go modiff and returns the formatted result string.
func Run(ctx context.Context, config *Config) (string, error) {
	if config == nil {
		return logErr(errNilConfig)
	}

	if config.repository == "" {
		return logErr(errNoRepository)
	}

	if config.from == config.to {
		return logErr(errSameFromTo)
	}

	err := validateConfig(config)
	if err != nil {
		return logErr(err)
	}

	dir, err := os.MkdirTemp("", "go-modiff")
	if err != nil {
		return logErr(err)
	}

	defer func() {
		err := os.RemoveAll(dir)
		if err != nil {
			logrus.Errorf("Failed to remove temp dir: %v", err)
		}
	}()

	err = cloneRepos(ctx, dir, config)
	if err != nil {
		return logErr(err)
	}

	mods, err := getModules(ctx, filepath.Join(dir, "from"), filepath.Join(dir, "to"))
	if err != nil {
		return "", err
	}

	result := diffModules(ctx, mods, config)

	switch config.format {
	case FormatJSON:
		applyFilter(&result, config.filter)

		return formatJSON(result)
	default:
		return formatMarkdown(result, config.link, config.headerLevel, config.filter), nil
	}
}

// CheckURLValid checks whether the given URL returns a valid (non-404) response.
func CheckURLValid(ctx context.Context, client http.Client, targetURL string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL, http.NoBody)
	if err != nil {
		return false, fmt.Errorf("error while creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error while sending request: %w", err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			logrus.Errorf("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return true, nil
}

func validateConfig(config *Config) error {
	if config.format != FormatMarkdown && config.format != FormatJSON {
		return errInvalidFormat
	}

	validFilters := map[string]bool{
		"": true, FilterAdded: true, FilterChanged: true, FilterRemoved: true,
	}

	if !validFilters[config.filter] {
		return errInvalidFilter
	}

	return nil
}

func toURL(name string) string {
	return "https://" + name
}

func logErr(err error) (string, error) {
	logrus.Error(err)

	return "", err
}

func goProxyURL() string {
	proxyEnv, exists := os.LookupEnv("GOPROXY")
	if !exists || proxyEnv == "" {
		return goProxyDefault
	}

	first := strings.SplitN(proxyEnv, ",", proxySplitParts)[0]
	if first == "direct" || first == "off" {
		return goProxyDefault
	}

	return first
}

func (info *goModInfo) isKnownHost() bool {
	return info.Origin.URL != "" &&
		(strings.HasPrefix(info.Origin.URL, "https://github.com/") ||
			strings.HasPrefix(info.Origin.URL, "https://go.googlesource.com/"))
}

func isGitHubModule(name string) bool {
	return strings.HasPrefix(name, "github.com/")
}

func gitHubBaseURL(name string) string {
	parts := strings.Split(name, "/")
	if len(parts) >= gitHubPathSegments {
		return "https://" + strings.Join(parts[:gitHubPathSegments], "/")
	}

	return "https://" + name
}

func (info *goModInfo) refName() string {
	if info.Origin.Ref == "" {
		return info.Origin.Hash
	}

	parts := strings.SplitN(info.Origin.Ref, "/", refSplitParts)
	if len(parts) == refSplitParts {
		return parts[refSplitParts-1]
	}

	return info.Origin.Ref
}

func (info *goModInfo) commitURL() string {
	if strings.HasPrefix(info.Origin.URL, "https://github.com/") {
		return fmt.Sprintf("%s/commit/%s", info.Origin.URL, info.Origin.Hash)
	}

	if strings.HasPrefix(info.Origin.URL, "https://go.googlesource.com/") {
		return fmt.Sprintf("%s/+/%s", info.Origin.URL, info.Origin.Hash)
	}

	return ""
}

func (info *goModInfo) compareURL(other *goModInfo) string {
	if strings.HasPrefix(info.Origin.URL, "https://github.com/") {
		return fmt.Sprintf(
			"%s/compare/%s...%s",
			info.Origin.URL, info.refName(), other.refName(),
		)
	}

	if strings.HasPrefix(info.Origin.URL, "https://go.googlesource.com/") {
		return fmt.Sprintf(
			"%s/+/%s^1..%s/",
			info.Origin.URL, info.Origin.Hash, other.Origin.Hash,
		)
	}

	return ""
}

func newHTTPClient() *http.Client {
	//nolint:exhaustruct // only Timeout needed
	return &http.Client{
		Timeout: time.Duration(httpTimeoutSeconds) * time.Second,
	}
}

func fetchModInfo(ctx context.Context, client *http.Client, module, version string) (goModInfo, error) {
	var info goModInfo

	infoURL := fmt.Sprintf("%s/%s/@v/%s.info", goProxyURL(), module, version)
	logrus.Debugf("Fetching module info from %s", infoURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, http.NoBody)
	if err != nil {
		return info, fmt.Errorf("creating proxy request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return info, fmt.Errorf("fetching module info from proxy: %w", err)
	}

	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			logrus.Errorf("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf(
			"%w %d for %s@%s",
			errProxyBadStatus, resp.StatusCode, module, version,
		)
	}

	err = json.NewDecoder(resp.Body).Decode(&info)
	if err != nil {
		return info, fmt.Errorf("decoding proxy response: %w", err)
	}

	return info, nil
}

func cloneRepos(ctx context.Context, dir string, config *Config) error {
	referenceDir := filepath.Join(dir, "reference")

	logrus.Infof("Cloning reference repository %s", config.repository)

	err := runGit(
		ctx, dir, "clone", "--filter=blob:none", "--bare",
		toURL(config.repository), referenceDir,
	)
	if err != nil {
		return err
	}

	logrus.Infof("Setting up 'from' at %s", config.from)

	err = cloneAtRevision(ctx, dir, referenceDir, config.repository, config.from, filepath.Join(dir, "from"))
	if err != nil {
		return err
	}

	logrus.Infof("Setting up 'to' at %s", config.to)

	return cloneAtRevision(ctx, dir, referenceDir, config.repository, config.to, filepath.Join(dir, "to"))
}

func cloneAtRevision(ctx context.Context, parentDir, referenceDir, repository, rev, targetDir string) error {
	err := runGit(
		ctx, parentDir, "clone", "--filter=blob:none",
		"--reference", referenceDir, "--no-checkout",
		toURL(repository), targetDir,
	)
	if err != nil {
		return err
	}

	return runGit(ctx, targetDir, "checkout", rev)
}

func prettifyVersion(version string) string {
	version = strings.TrimSuffix(version, "+incompatible")

	versionSplit := strings.Split(version, "-")
	if len(versionSplit) < minPseudoVersionParts {
		return version
	}

	hash := versionSplit[len(versionSplit)-1]
	if len(hash) > shortHashLength {
		return hash[:shortHashLength]
	}

	// This should never happen but who knows what go modules will do next.
	return hash
}

func classifyModule(
	mod entry, name string, beforeInfo, afterInfo *goModInfo, addLinks bool,
) (string, ModuleChange) {
	beforeDisplay := prettifyVersion(mod.beforeVersion)
	afterDisplay := prettifyVersion(mod.afterVersion)

	if mod.beforeVersion == "" {
		change := ModuleChange{Name: name, Before: "", After: afterDisplay, Link: ""}
		if addLinks {
			change.Link = generateSingleLink(afterDisplay, afterInfo, name)
		}

		return FilterAdded, change
	}

	if mod.afterVersion == "" {
		change := ModuleChange{Name: name, Before: beforeDisplay, After: "", Link: ""}
		if addLinks {
			change.Link = generateSingleLink(beforeDisplay, beforeInfo, name)
		}

		return FilterRemoved, change
	}

	if mod.beforeVersion != mod.afterVersion {
		change := ModuleChange{Name: name, Before: beforeDisplay, After: afterDisplay, Link: ""}
		if addLinks {
			change.Link = generateCompareLink(beforeDisplay, afterDisplay, beforeInfo, afterInfo, name)
		}

		return FilterChanged, change
	}

	return "", ModuleChange{Name: "", Before: "", After: "", Link: ""}
}

func generateSingleLink(display string, info *goModInfo, name string) string {
	if info != nil && info.isKnownHost() {
		return info.commitURL()
	}

	if isGitHubModule(name) {
		return fmt.Sprintf("%s/tree/%s", gitHubBaseURL(name), display)
	}

	return ""
}

func generateCompareLink(
	beforeDisplay, afterDisplay string, beforeInfo, afterInfo *goModInfo, name string,
) string {
	if beforeInfo != nil && afterInfo != nil && beforeInfo.isKnownHost() {
		return beforeInfo.compareURL(afterInfo)
	}

	if isGitHubModule(name) {
		return fmt.Sprintf(
			"%s/compare/%s...%s",
			gitHubBaseURL(name), beforeDisplay, afterDisplay,
		)
	}

	return ""
}

func diffModules(ctx context.Context, mods modules, config *Config) DiffResult {
	type moduleResult struct {
		category string
		change   ModuleChange
	}

	results := make([]moduleResult, 0, len(mods))

	if config.link {
		logrus.Infof("Fetching module info for %d modules", len(mods))

		var (
			mutex     sync.Mutex
			waitGrp   sync.WaitGroup
			semaphore = make(chan struct{}, max(config.concurrency, 1))
		)

		client := newHTTPClient()

		for name, mod := range mods {
			waitGrp.Go(func() {
				semaphore <- struct{}{}

				beforeInfo, afterInfo := fetchModInfoPair(ctx, client, name, mod)

				<-semaphore

				category, change := classifyModule(mod, name, beforeInfo, afterInfo, true)
				if category == "" {
					return
				}

				mutex.Lock()

				results = append(results, moduleResult{category: category, change: change})
				mutex.Unlock()
			})
		}

		waitGrp.Wait()
	} else {
		for name, mod := range mods {
			category, change := classifyModule(mod, name, nil, nil, false)
			if category == "" {
				continue
			}

			results = append(results, moduleResult{category: category, change: change})
		}
	}

	diffResult := DiffResult{
		Added:   []ModuleChange{},
		Changed: []ModuleChange{},
		Removed: []ModuleChange{},
	}

	for _, res := range results {
		switch res.category {
		case FilterAdded:
			diffResult.Added = append(diffResult.Added, res.change)
		case FilterChanged:
			diffResult.Changed = append(diffResult.Changed, res.change)
		case FilterRemoved:
			diffResult.Removed = append(diffResult.Removed, res.change)
		}
	}

	sort.Slice(diffResult.Added, func(i, j int) bool { return diffResult.Added[i].Name < diffResult.Added[j].Name })
	sort.Slice(diffResult.Changed, func(i, j int) bool { return diffResult.Changed[i].Name < diffResult.Changed[j].Name })
	sort.Slice(diffResult.Removed, func(i, j int) bool { return diffResult.Removed[i].Name < diffResult.Removed[j].Name })

	logrus.Infof("%d modules added", len(diffResult.Added))
	logrus.Infof("%d modules changed", len(diffResult.Changed))
	logrus.Infof("%d modules removed", len(diffResult.Removed))

	return diffResult
}

func applyFilter(result *DiffResult, filter string) {
	switch filter {
	case FilterAdded:
		result.Changed = []ModuleChange{}
		result.Removed = []ModuleChange{}
	case FilterChanged:
		result.Added = []ModuleChange{}
		result.Removed = []ModuleChange{}
	case FilterRemoved:
		result.Added = []ModuleChange{}
		result.Changed = []ModuleChange{}
	}
}

func fetchModInfoPair(ctx context.Context, client *http.Client, name string, mod entry) (*goModInfo, *goModInfo) {
	var beforeInfo, afterInfo *goModInfo

	if mod.beforeVersion != "" {
		info, err := fetchModInfo(ctx, client, name, mod.beforeVersion)
		if err != nil {
			logrus.Debugf("Could not fetch module info for %s@%s: %v", name, mod.beforeVersion, err)
		} else {
			beforeInfo = &info
		}
	}

	if mod.afterVersion != "" {
		info, err := fetchModInfo(ctx, client, name, mod.afterVersion)
		if err != nil {
			logrus.Debugf("Could not fetch module info for %s@%s: %v", name, mod.afterVersion, err)
		} else {
			afterInfo = &info
		}
	}

	return beforeInfo, afterInfo
}

func formatModuleMarkdown(change ModuleChange, category string, addLinks bool) string {
	txt := fmt.Sprintf("- %s: ", change.Name)

	switch category {
	case FilterAdded:
		if addLinks && change.Link != "" {
			txt += fmt.Sprintf("[%s](%s)", change.After, change.Link)
		} else {
			txt += change.After
		}
	case FilterRemoved:
		if addLinks && change.Link != "" {
			txt += fmt.Sprintf("[%s](%s)", change.Before, change.Link)
		} else {
			txt += change.Before
		}
	case FilterChanged:
		if addLinks && change.Link != "" {
			txt += fmt.Sprintf("[%s → %s](%s)", change.Before, change.After, change.Link)
		} else {
			txt += fmt.Sprintf("%s → %s", change.Before, change.After)
		}
	}

	return txt
}

func formatMarkdown(result DiffResult, addLinks bool, headerLevel uint, filter string) string {
	level := min(headerLevel, maxHeaderLevel)
	builder := &strings.Builder{}

	fmt.Fprintf(
		builder, "%s Dependencies\n", strings.Repeat("#", int(level)), //nolint:gosec // level is clamped to maxHeaderLevel
	)

	writeSection := func(section string, changes []ModuleChange, category string) {
		fmt.Fprintf(
			builder,
			"\n%s %s\n", strings.Repeat("#", int(level)+1), section, //nolint:gosec // level is clamped to maxHeaderLevel
		)

		if len(changes) > 0 {
			for _, change := range changes {
				fmt.Fprintf(builder, "%s\n", formatModuleMarkdown(change, category, addLinks))
			}
		} else {
			builder.WriteString("_Nothing has changed._\n")
		}
	}

	if filter == "" || filter == FilterAdded {
		writeSection("Added", result.Added, FilterAdded)
	}

	if filter == "" || filter == FilterChanged {
		writeSection("Changed", result.Changed, FilterChanged)
	}

	if filter == "" || filter == FilterRemoved {
		writeSection("Removed", result.Removed, FilterRemoved)
	}

	return builder.String()
}

func formatJSON(result DiffResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling JSON result: %w", err)
	}

	return string(data) + "\n", nil
}

func parseModuleLine(line string) (string, string, bool) {
	split := strings.Split(line, " ")
	if len(split) < minModuleFields {
		return "", "", false
	}

	// Rewrites have to be handled differently.
	if len(split) > minModuleFields && split[2] == "=>" {
		// Local rewrites without any version will be skipped.
		if len(split) == localRewriteFields {
			return "", "", false
		}

		// Use the rewritten version and name if available.
		if len(split) == fullRewriteFields {
			split[0] = split[3]
			split[1] = split[4]
		}
	}

	modName := strings.TrimSpace(split[0])
	modVersion := strings.TrimSpace(split[1])

	return modName, modVersion, true
}

func toLineSet(input string) map[string]bool {
	lines := make(map[string]bool)

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		lines[scanner.Text()] = true
	}

	return lines
}

func getModules(ctx context.Context, fromDir, toDir string) (modules, error) {
	before, err := retrieveModules(ctx, fromDir)
	if err != nil {
		return nil, err
	}

	after, err := retrieveModules(ctx, toDir)
	if err != nil {
		return nil, err
	}

	beforeLines := toLineSet(before)
	afterLines := toLineSet(after)

	logrus.Info("Processing module diffs")

	result := modules{}

	parseInto := func(input string, skipLines map[string]bool, apply func(result *entry, version string)) {
		scanner := bufio.NewScanner(strings.NewReader(input))
		for scanner.Scan() {
			line := scanner.Text()

			// Skip lines present in both lists (unchanged).
			if skipLines[line] {
				logrus.Debugf("Skipping unchanged module: %s", line)

				continue
			}

			name, version, ok := parseModuleLine(line)
			if !ok {
				continue
			}

			modEntry := new(entry)
			if existing, found := result[name]; found {
				modEntry = &existing
			}

			apply(modEntry, version)
			result[name] = *modEntry
		}
	}

	parseInto(before, afterLines, func(result *entry, version string) { result.beforeVersion = version })
	parseInto(after, beforeLines, func(result *entry, version string) { result.afterVersion = version })

	logrus.Infof("%d modules found", len(result))

	return result, nil
}

func retrieveModules(ctx context.Context, workDir string) (string, error) {
	logrus.Debugf("Listing modules in %s", workDir)

	mods, err := runCmdOutput(ctx, workDir, "go", "list", "-mod=readonly", "-m", "all")
	if err != nil {
		logrus.Error(err)

		return "", err
	}

	return strings.TrimSpace(string(mods)), nil
}

func runGit(ctx context.Context, dir string, args ...string) error {
	return runCmd(ctx, dir, "git", args...)
}

func runCmd(ctx context.Context, dir, cmd string, args ...string) error {
	_, err := runCmdOutput(ctx, dir, cmd, args...)

	return err
}

func runCmdOutput(ctx context.Context, dir, cmd string, args ...string) ([]byte, error) {
	//nolint:gosec // cmd is always controlled internally
	command := exec.CommandContext(ctx, cmd, args...)
	command.Dir = dir

	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError

		stderr := []byte{}
		if errors.As(err, &exitError) {
			stderr = exitError.Stderr
		}

		return nil, fmt.Errorf(
			"unable to run cmd: %s %s, workdir: %s, stdout: %s, stderr: %v, error: %w",
			cmd,
			strings.Join(args, " "),
			dir,
			string(output),
			string(stderr),
			err,
		)
	}

	return output, nil
}
