package modiff

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
)

type moduleResult struct {
	category string
	change   ModuleChange
}

func diffModules(ctx context.Context, mods modules, config *Config) DiffResult {
	results := make([]moduleResult, 0, len(mods))

	if config.link {
		slog.Info("Fetching module info", "modules", len(mods))

		var (
			mutex     sync.Mutex
			waitGrp   sync.WaitGroup
			semaphore = make(chan struct{}, max(config.concurrency, 1))
		)

		client := newHTTPClient()

		for name, mod := range mods {
			waitGrp.Go(func() {
				beforeInfo, afterInfo := fetchModInfoPair(ctx, client, name, mod, semaphore)

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

	return buildDiffResult(results)
}

func buildDiffResult(results []moduleResult) DiffResult {
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

	slices.SortFunc(diffResult.Added, func(a, b ModuleChange) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(diffResult.Changed, func(a, b ModuleChange) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(diffResult.Removed, func(a, b ModuleChange) int {
		return strings.Compare(a.Name, b.Name)
	})

	slog.Info("Modules added", "count", len(diffResult.Added))
	slog.Info("Modules changed", "count", len(diffResult.Changed))
	slog.Info("Modules removed", "count", len(diffResult.Removed))

	return diffResult
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
			change.Link = generateCompareLink(
				beforeDisplay,
				afterDisplay,
				beforeInfo,
				afterInfo,
				name,
			)
		}

		return FilterChanged, change
	}

	return "", ModuleChange{Name: "", Before: "", After: "", Link: ""}
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
	if beforeInfo != nil && afterInfo != nil &&
		beforeInfo.isKnownHost() && afterInfo.isKnownHost() {
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
