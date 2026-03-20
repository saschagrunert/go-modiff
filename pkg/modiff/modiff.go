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
)

var (
	errNilConfig      = errors.New("config is nil")
	errNoRepository   = errors.New("repository is required")
	errSameFromTo     = errors.New("no diff possible if `from` equals `to`")
	errProxyBadStatus = errors.New("proxy returned unexpected status")
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

// Config is the structure passed to `Run`.
type Config struct {
	repository  string
	from        string
	to          string
	link        bool
	headerLevel uint
}

// NewConfig creates a new configuration.
func NewConfig(repository, from, to string, link bool, headerLevel uint) *Config {
	return &Config{repository, from, to, link, headerLevel}
}

// Run starts go modiff and returns the markdown string.
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

	return diffModules(ctx, mods, config.link, config.headerLevel), nil
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

func fetchModInfo(ctx context.Context, module, version string) (goModInfo, error) {
	var info goModInfo

	infoURL := fmt.Sprintf("%s/%s/@v/%s.info", goProxyURL(), module, version)
	logrus.Debugf("Fetching module info from %s", infoURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, http.NoBody)
	if err != nil {
		return info, fmt.Errorf("creating proxy request: %w", err)
	}

	client := http.Client{} //nolint:exhaustruct // zero value client is intentional

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

func classifyModule(mod entry, name string, addLinks bool, beforeInfo, afterInfo *goModInfo) (string, string) {
	beforeDisplay := prettifyVersion(mod.beforeVersion)
	afterDisplay := prettifyVersion(mod.afterVersion)
	txt := fmt.Sprintf("- %s: ", name)

	if mod.beforeVersion == "" { //nolint:gocritic // if-else chain is clearer here
		txt += formatSingle(afterDisplay, afterInfo, name, addLinks)

		return "added", txt
	} else if mod.afterVersion == "" {
		txt += formatSingle(beforeDisplay, beforeInfo, name, addLinks)

		return "removed", txt
	} else if mod.beforeVersion != mod.afterVersion {
		txt += formatChanged(beforeDisplay, afterDisplay, beforeInfo, afterInfo, name, addLinks)

		return "changed", txt
	}

	return "", ""
}

func formatSingle(display string, info *goModInfo, name string, addLinks bool) string {
	if !addLinks {
		return display
	}

	if info != nil && info.isKnownHost() {
		return fmt.Sprintf("[%s](%s)", display, info.commitURL())
	}

	if isGitHubModule(name) {
		return fmt.Sprintf("[%s](%s/tree/%s)", display, gitHubBaseURL(name), display)
	}

	return display
}

func formatChanged(
	beforeDisplay, afterDisplay string, beforeInfo, afterInfo *goModInfo, name string, addLinks bool,
) string {
	if !addLinks {
		return fmt.Sprintf("%s → %s", beforeDisplay, afterDisplay)
	}

	if beforeInfo != nil && afterInfo != nil && beforeInfo.isKnownHost() {
		return fmt.Sprintf("[%s → %s](%s)", beforeDisplay, afterDisplay, beforeInfo.compareURL(afterInfo))
	}

	if isGitHubModule(name) {
		return fmt.Sprintf(
			"[%s → %s](%s/compare/%s...%s)",
			beforeDisplay, afterDisplay, gitHubBaseURL(name), beforeDisplay, afterDisplay,
		)
	}

	return fmt.Sprintf("%s → %s", beforeDisplay, afterDisplay)
}

type classifiedModule struct {
	category string
	txt      string
}

func diffModules(ctx context.Context, mods modules, addLinks bool, headerLevel uint) string {
	results := make(map[string]classifiedModule, len(mods))

	if addLinks {
		var (
			mutex   sync.Mutex
			waitGrp sync.WaitGroup
			//nolint:mnd // reasonable concurrency limit for HTTP requests
			semaphore = make(chan struct{}, 10)
		)

		for name, mod := range mods {
			waitGrp.Go(func() {
				semaphore <- struct{}{}

				beforeInfo, afterInfo := fetchModInfoPair(ctx, name, mod)

				<-semaphore

				category, txt := classifyModule(mod, name, addLinks, beforeInfo, afterInfo)

				mutex.Lock()
				results[name] = classifiedModule{category, txt}
				mutex.Unlock()
			})
		}

		waitGrp.Wait()
	} else {
		for name, mod := range mods {
			category, txt := classifyModule(mod, name, addLinks, nil, nil)
			results[name] = classifiedModule{category, txt}
		}
	}

	var added, removed, changed []string

	for _, res := range results {
		switch res.category {
		case "added":
			added = append(added, res.txt)
		case "removed":
			removed = append(removed, res.txt)
		case "changed":
			changed = append(changed, res.txt)
		}
	}

	sort.Strings(added)
	sort.Strings(changed)
	sort.Strings(removed)
	logrus.Infof("%d modules added", len(added))
	logrus.Infof("%d modules changed", len(changed))
	logrus.Infof("%d modules removed", len(removed))

	return formatMarkdown(added, changed, removed, headerLevel)
}

func fetchModInfoPair(ctx context.Context, name string, mod entry) (*goModInfo, *goModInfo) {
	var beforeInfo, afterInfo *goModInfo

	if mod.beforeVersion != "" {
		info, err := fetchModInfo(ctx, name, mod.beforeVersion)
		if err != nil {
			logrus.Debugf("Could not fetch module info for %s@%s: %v", name, mod.beforeVersion, err)
		} else {
			beforeInfo = &info
		}
	}

	if mod.afterVersion != "" {
		info, err := fetchModInfo(ctx, name, mod.afterVersion)
		if err != nil {
			logrus.Debugf("Could not fetch module info for %s@%s: %v", name, mod.afterVersion, err)
		} else {
			afterInfo = &info
		}
	}

	return beforeInfo, afterInfo
}

func formatMarkdown(added, changed, removed []string, headerLevel uint) string {
	level := min(headerLevel, maxHeaderLevel)
	builder := &strings.Builder{}

	fmt.Fprintf(
		builder, "%s Dependencies\n", strings.Repeat("#", int(level)), //nolint:gosec // level is clamped to maxHeaderLevel
	)

	writeSection := func(section string, input []string) {
		fmt.Fprintf(
			builder,
			"\n%s %s\n", strings.Repeat("#", int(level)+1), section, //nolint:gosec // level is clamped to maxHeaderLevel
		)

		if len(input) > 0 {
			for _, mod := range input {
				fmt.Fprintf(builder, "%s\n", mod)
			}
		} else {
			builder.WriteString("_Nothing has changed._\n")
		}
	}

	writeSection("Added", added)
	writeSection("Changed", changed)
	writeSection("Removed", removed)

	return builder.String()
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
	command.Stderr = nil
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
