// Package modiff provides functionality to diff Go module dependencies
// between two git revisions of a repository.
package modiff

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// gitHubPathSegments is the number of path segments in a standard
	// GitHub module path (e.g. github.com/owner/repo).
	gitHubPathSegments = 3

	// gitHubSubpathSegments is the number of path segments when a module
	// has an additional sub-path (e.g. github.com/owner/repo/subpath).
	gitHubSubpathSegments = 4

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
)

var (
	errNilConfig    = errors.New("config is nil")
	errNoRepository = errors.New("repository is required")
	errSameFromTo   = errors.New("no diff possible if `from` equals `to`")
)

type entry struct {
	beforeVersion string
	afterVersion  string
	linkPrefix    string
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

	logrus.Infof("Setting up repository %s", config.repository)

	err = runGit(ctx, dir, "init")
	if err != nil {
		return logErr(err)
	}

	err = runGit(ctx, dir, "remote", "add", "origin", toURL(config.repository))
	if err != nil {
		return logErr(err)
	}

	mods, err := getModules(ctx, dir, config.from, config.to)
	if err != nil {
		return "", err
	}

	return diffModules(mods, config.link, config.headerLevel), nil
}

func toURL(name string) string {
	return "https://" + name
}

func isGitHubURL(name string) bool {
	return strings.HasPrefix(name, "github.com")
}

func sanitizeTag(tag string) string {
	return strings.TrimSuffix(tag, "+incompatible")
}

func logErr(err error) (string, error) {
	logrus.Error(err)

	return "", err
}

func buildTreeLink(splitPrefix []string, version string) string {
	prefixWithTree := strings.Join(splitPrefix, "/") + "/tree"
	if len(splitPrefix) >= gitHubPathSegments {
		prefixWithTree = strings.Join(slices.Insert(splitPrefix, gitHubPathSegments, "tree"), "/")
	}

	return fmt.Sprintf("[%s](%s/%s)", version, toURL(prefixWithTree), sanitizeTag(version))
}

func buildCompareLink(mod entry, splitLinkPrefix []string) string {
	prefixWithCompare := fmt.Sprintf("%s/%s", mod.linkPrefix, "compare")
	afterVersion := sanitizeTag(mod.afterVersion)

	if len(splitLinkPrefix) > gitHubPathSegments {
		prefixWithCompare = strings.Join(slices.Insert(splitLinkPrefix, gitHubPathSegments, "compare"), "/")
		afterVersion = fmt.Sprintf("%s/%s", strings.Join(splitLinkPrefix[gitHubPathSegments:], "/"), afterVersion)
	}

	return fmt.Sprintf("[%s → %s](%s/%s...%s)",
		mod.beforeVersion, mod.afterVersion, toURL(prefixWithCompare),
		sanitizeTag(mod.beforeVersion), afterVersion)
}

func classifyModule(mod entry, name string, addLinks bool) (string, string) {
	txt := fmt.Sprintf("- %s: ", name)
	splitLinkPrefix := strings.Split(mod.linkPrefix, "/")

	if mod.beforeVersion == "" { //nolint:gocritic // if-else chain is clearer here
		if addLinks && isGitHubURL(mod.linkPrefix) {
			txt += buildTreeLink(splitLinkPrefix, mod.afterVersion)
		} else {
			txt += mod.afterVersion
		}

		return "added", txt
	} else if mod.afterVersion == "" {
		if addLinks && isGitHubURL(mod.linkPrefix) {
			txt += buildTreeLink(splitLinkPrefix, mod.beforeVersion)
		} else {
			txt += mod.beforeVersion
		}

		return "removed", txt
	} else if mod.beforeVersion != mod.afterVersion {
		if addLinks && isGitHubURL(mod.linkPrefix) {
			txt += buildCompareLink(mod, splitLinkPrefix)
		} else {
			txt += fmt.Sprintf("%s → %s", mod.beforeVersion, mod.afterVersion)
		}

		return "changed", txt
	}

	return "", ""
}

func diffModules(mods modules, addLinks bool, headerLevel uint) string {
	var added, removed, changed []string

	for name, mod := range mods {
		category, txt := classifyModule(mod, name, addLinks)

		switch category {
		case "added":
			added = append(added, txt)
		case "removed":
			removed = append(removed, txt)
		case "changed":
			changed = append(changed, txt)
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

func resolveLinkPrefix(ctx context.Context, name, version string) string {
	linkPrefix := name

	splitLink := strings.Split(linkPrefix, "/")
	if len(splitLink) != gitHubSubpathSegments {
		return linkPrefix
	}

	// Check if the last part of the string is part of the tag.
	linkPrefixTree := strings.Join(slices.Insert(splitLink, gitHubPathSegments, "tree"), "/")
	checkURL := fmt.Sprintf("https://%s/%s", linkPrefixTree, strings.TrimSpace(version))

	client := http.Client{} //nolint:exhaustruct // zero value client is intentional
	valid, err := CheckURLValid(ctx, client, checkURL)

	if !valid && err == nil {
		linkPrefix = strings.Join(splitLink[:gitHubPathSegments], "/")
	}

	return linkPrefix
}

func prettifyVersion(version string) string {
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

func parseModuleLine(ctx context.Context, line string) (string, string, string, bool) {
	split := strings.Split(line, " ")
	if len(split) < minModuleFields {
		return "", "", "", false
	}

	// Rewrites have to be handled differently.
	if len(split) > minModuleFields && split[2] == "=>" {
		// Local rewrites without any version will be skipped.
		if len(split) == localRewriteFields {
			return "", "", "", false
		}

		// Use the rewritten version and name if available.
		if len(split) == fullRewriteFields {
			split[0] = split[3]
			split[1] = split[4]
		}
	}

	modName := strings.TrimSpace(split[0])
	modLinkPrefix := resolveLinkPrefix(ctx, modName, split[1])
	modVersion := prettifyVersion(strings.TrimSpace(split[1]))

	return modName, modLinkPrefix, modVersion, true
}

func getModules(ctx context.Context, workDir, fromRev, toRev string) (modules, error) {
	before, err := retrieveModules(ctx, fromRev, workDir)
	if err != nil {
		return nil, err
	}

	after, err := retrieveModules(ctx, toRev, workDir)
	if err != nil {
		return nil, err
	}

	result := modules{}

	parseInto := func(input string, apply func(result *entry, version string)) {
		scanner := bufio.NewScanner(strings.NewReader(input))
		for scanner.Scan() {
			name, linkPrefix, version, ok := parseModuleLine(ctx, scanner.Text())
			if !ok {
				continue
			}

			modEntry := new(entry)
			if existing, found := result[name]; found {
				modEntry = &existing
			}

			apply(modEntry, version)
			modEntry.linkPrefix = linkPrefix
			result[name] = *modEntry
		}
	}

	parseInto(before, func(result *entry, version string) { result.beforeVersion = version })
	parseInto(after, func(result *entry, version string) { result.afterVersion = version })

	logrus.Infof("%d modules found", len(result))

	return result, nil
}

// CheckURLValid checks whether the given URL returns a valid (non-404) response.
func CheckURLValid(ctx context.Context, client http.Client, targetURL string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, http.NoBody)
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

func retrieveModules(ctx context.Context, rev, workDir string) (string, error) {
	logrus.Infof("Retrieving modules of %s", rev)

	err := runGit(ctx, workDir, "fetch", "--depth=1", "origin", rev)
	if err != nil {
		logrus.Error(err)

		return "", err
	}

	err = runGit(ctx, workDir, "checkout", "-f", "FETCH_HEAD")
	if err != nil {
		logrus.Error(err)

		return "", err
	}

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
