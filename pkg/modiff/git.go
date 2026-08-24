package modiff

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
)

func cloneRepos(ctx context.Context, dir string, config *Config) error {
	referenceDir := filepath.Join(dir, "reference")
	repoURL := "https://" + config.repository

	slog.Info("Cloning reference repository", "repository", config.repository)

	err := runGit(
		ctx, dir, "clone", "--filter=blob:none", "--bare",
		repoURL, referenceDir,
	)
	if err != nil {
		return fmt.Errorf("cloning repository %s: %w", config.repository, err)
	}

	slog.Info("Setting up revision", "rev", config.from)

	err = cloneAtRevision(ctx, dir, referenceDir, repoURL, config.from, filepath.Join(dir, "from"))
	if err != nil {
		return fmt.Errorf("setting up revision %s: %w", config.from, err)
	}

	slog.Info("Setting up revision", "rev", config.to)

	err = cloneAtRevision(ctx, dir, referenceDir, repoURL, config.to, filepath.Join(dir, "to"))
	if err != nil {
		return fmt.Errorf("setting up revision %s: %w", config.to, err)
	}

	return nil
}

func cloneAtRevision(
	ctx context.Context,
	parentDir, referenceDir, repoURL, rev, targetDir string,
) error {
	err := runGit(
		ctx, parentDir, "clone", "--filter=blob:none",
		"--reference", referenceDir, "--no-checkout",
		repoURL, targetDir,
	)
	if err != nil {
		return err
	}

	return runGit(ctx, targetDir, "checkout", rev)
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

	slog.Info("Processing module diffs")

	beforeMods := parseAllModules(before)
	afterMods := parseAllModules(after)
	result := modules{}

	for name, beforeVer := range beforeMods {
		afterVer, inAfter := afterMods[name]
		if !inAfter {
			result[name] = entry{beforeVersion: beforeVer, afterVersion: ""}
		} else if beforeVer != afterVer {
			result[name] = entry{beforeVersion: beforeVer, afterVersion: afterVer}
		}
	}

	for name, afterVer := range afterMods {
		if _, inBefore := beforeMods[name]; !inBefore {
			result[name] = entry{beforeVersion: "", afterVersion: afterVer}
		}
	}

	slog.Info("Modules found", "count", len(result))

	return result, nil
}

func retrieveModules(ctx context.Context, workDir string) (string, error) {
	slog.Debug("Listing modules", "dir", workDir)

	mods, err := runCmdOutput(ctx, workDir, "go", "list", "-mod=readonly", "-m", "all")
	if err != nil {
		return "", fmt.Errorf("listing modules: %w", err)
	}

	return strings.TrimSpace(string(mods)), nil
}

func parseModuleLine(line string) (string, string, bool) {
	split := strings.Split(line, " ")
	if len(split) < minModuleFields {
		return "", "", false
	}

	if len(split) > minModuleFields && split[2] == "=>" {
		if len(split) == localRewriteFields {
			return "", "", false
		}

		if len(split) == fullRewriteFields {
			split[0] = split[3]
			split[1] = split[4]
		}
	}

	modName := strings.TrimSpace(split[0])
	modVersion := strings.TrimSpace(split[1])

	return modName, modVersion, true
}

func parseAllModules(input string) map[string]string {
	result := make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		name, version, ok := parseModuleLine(scanner.Text())
		if ok {
			result[name] = version
		}
	}

	return result
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
		var stderrStr string

		if exitError, ok := errors.AsType[*exec.ExitError](err); ok {
			stderrStr = strings.TrimSpace(string(exitError.Stderr))
		}

		if stderrStr != "" {
			return nil, fmt.Errorf(
				"running %s %s: %s: %w",
				cmd, strings.Join(args, " "), stderrStr, err,
			)
		}

		return nil, fmt.Errorf(
			"running %s %s: %w",
			cmd, strings.Join(args, " "), err,
		)
	}

	return output, nil
}
