package modiff

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type goModOrigin struct {
	VCS  string `json:"VCS"`  //nolint:tagliatelle // matches proxy response format
	URL  string `json:"URL"`  //nolint:tagliatelle // matches proxy response format
	Hash string `json:"Hash"` //nolint:tagliatelle // matches proxy response format
	Ref  string `json:"Ref"`  //nolint:tagliatelle // matches proxy response format
}

type goModInfo struct {
	Version string      `json:"Version"` //nolint:tagliatelle // matches proxy response format
	Origin  goModOrigin `json:"Origin"`  //nolint:tagliatelle // matches proxy response format
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

func goProxyURL() string {
	proxyEnv, exists := os.LookupEnv("GOPROXY")
	if !exists || proxyEnv == "" {
		return goProxyDefault
	}

	first, _, _ := strings.Cut(proxyEnv, ",")
	first, _, _ = strings.Cut(first, "|")

	if first == "direct" || first == "off" {
		return goProxyDefault
	}

	return first
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Duration(httpTimeoutSeconds) * time.Second,
	}
}

func fetchModInfo(
	ctx context.Context, client *http.Client, module, version string,
) (goModInfo, error) {
	var info goModInfo

	infoURL := fmt.Sprintf("%s/%s/@v/%s.info", goProxyURL(), module, version)
	slog.Debug("Fetching module info", "url", infoURL)

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
			slog.Error("Failed to close response body", "error", closeErr)
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

func fetchModInfoPair(
	ctx context.Context,
	client *http.Client,
	name string,
	mod entry,
	semaphore chan struct{},
) (*goModInfo, *goModInfo) {
	var (
		beforeInfo, afterInfo *goModInfo
		waitGrp               sync.WaitGroup
	)

	if mod.beforeVersion != "" {
		waitGrp.Go(func() {
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

			info, err := fetchModInfo(ctx, client, name, mod.beforeVersion)

			<-semaphore

			if err != nil {
				slog.Debug("Could not fetch module info",
					"module", name,
					"version", mod.beforeVersion,
					"error", err,
				)
			} else {
				beforeInfo = &info
			}
		})
	}

	if mod.afterVersion != "" {
		waitGrp.Go(func() {
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

			info, err := fetchModInfo(ctx, client, name, mod.afterVersion)

			<-semaphore

			if err != nil {
				slog.Debug("Could not fetch module info",
					"module", name,
					"version", mod.afterVersion,
					"error", err,
				)
			} else {
				afterInfo = &info
			}
		})
	}

	waitGrp.Wait()

	return beforeInfo, afterInfo
}
