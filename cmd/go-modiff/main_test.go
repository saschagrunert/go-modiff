package main

import (
	"testing"

	"github.com/urfave/cli/v3"

	"github.com/saschagrunert/go-modiff/pkg/modiff"
)

func TestBuildApp(test *testing.T) {
	test.Parallel()

	app := buildApp()

	if app.Name != "go-modiff" {
		test.Errorf("expected app name 'go-modiff', got %q", app.Name)
	}

	if !app.EnableShellCompletion {
		test.Error("expected shell completion to be enabled")
	}

	if app.Action == nil {
		test.Error("expected action to be set")
	}
}

func TestBuildFlags(test *testing.T) {
	test.Parallel()

	flags := buildFlags()

	expectedFlags := map[string]bool{
		repositoryArg:  false,
		fromArg:        false,
		toArg:          false,
		linkArg:        false,
		headerLevelArg: false,
		formatArg:      false,
		filterArg:      false,
		concurrencyArg: false,
		debugFlag:      false,
	}

	for _, flag := range flags {
		for _, name := range flag.Names() {
			if _, ok := expectedFlags[name]; ok {
				expectedFlags[name] = true
			}
		}
	}

	for name, found := range expectedFlags {
		if !found {
			test.Errorf("expected flag %q not found", name)
		}
	}
}

func TestBuildFlagDefaults(test *testing.T) {
	test.Parallel()

	app := buildApp()

	var toDefault string

	var formatDefault string

	var concurrencyDefault uint

	for _, flag := range app.Flags {
		for _, name := range flag.Names() {
			switch name {
			case toArg:
				if sf, ok := flag.(*cli.StringFlag); ok {
					toDefault = sf.Value
				}
			case formatArg:
				if sf, ok := flag.(*cli.StringFlag); ok {
					formatDefault = sf.Value
				}
			case concurrencyArg:
				if uf, ok := flag.(*cli.UintFlag); ok {
					concurrencyDefault = uf.Value
				}
			}
		}
	}

	if toDefault != "HEAD" {
		test.Errorf("expected 'to' default 'HEAD', got %q", toDefault)
	}

	if formatDefault != modiff.FormatMarkdown {
		test.Errorf("expected 'format' default %q, got %q", modiff.FormatMarkdown, formatDefault)
	}

	if concurrencyDefault != modiff.DefaultConcurrency {
		test.Errorf(
			"expected 'concurrency' default %d, got %d",
			modiff.DefaultConcurrency, concurrencyDefault,
		)
	}
}

func TestRunMissingFrom(test *testing.T) {
	test.Parallel()

	app := buildApp()

	err := app.Run(test.Context(), []string{"go-modiff", "--repository", "github.com/foo/bar"})
	if err == nil {
		test.Error("expected error for missing --from flag")
	}
}
