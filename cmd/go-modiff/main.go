// Package main provides the go-modiff CLI entry point.
package main

import (
	"context"
	"fmt"
	"os"

	ccli "github.com/saschagrunert/ccli/v3"
	"github.com/saschagrunert/go-modiff/pkg/modiff"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

const (
	repositoryArg  = "repository"
	fromArg        = "from"
	toArg          = "to"
	linkArg        = "link"
	headerLevelArg = "header-level"
	formatArg      = "format"
	filterArg      = "filter"
	concurrencyArg = "concurrency"
	debugFlag      = "debug"
)

func main() {
	app := buildApp()

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		os.Exit(1)
	}
}

func buildApp() *cli.Command {
	app := ccli.NewCommand()
	app.Name = "go-modiff"
	app.Version = "1.4.0"
	app.Authors = []any{"Sascha Grunert <mail@saschagrunert.de>"}
	app.Usage = "Command line tool for diffing go module " +
		"dependency changes between versions"
	app.UsageText = app.Usage
	app.Flags = buildFlags()
	app.Commands = buildCommands()
	app.Action = run

	return app
}

func buildFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    repositoryArg,
			Aliases: []string{"r"},
			Usage:   "repository to be used, like: github.com/owner/repo",
		},
		&cli.StringFlag{
			Name:    fromArg,
			Aliases: []string{"f"},
			Value:   "HEAD",
			Usage:   "the start of the comparison, any valid git rev",
		},
		&cli.StringFlag{
			Name:    toArg,
			Aliases: []string{"t"},
			Value:   "HEAD",
			Usage:   "the end of the comparison, any valid git rev",
		},
		&cli.BoolFlag{
			Name:    linkArg,
			Aliases: []string{"l"},
			Usage:   "add diff links to the markdown output",
		},
		&cli.UintFlag{
			Name:    headerLevelArg,
			Aliases: []string{"i"},
			Value:   1,
			Usage:   "markdown header level depth",
		},
		&cli.StringFlag{
			Name:    formatArg,
			Aliases: []string{"o"},
			Value:   modiff.FormatMarkdown,
			Usage:   "output format (markdown or json)",
		},
		&cli.StringFlag{
			Name:  filterArg,
			Usage: "filter output by category (added, changed, or removed)",
		},
		&cli.UintFlag{
			Name:    concurrencyArg,
			Aliases: []string{"c"},
			Value:   modiff.DefaultConcurrency,
			Usage:   "number of concurrent proxy requests for link resolution",
		},
		&cli.BoolFlag{
			Name:    debugFlag,
			Aliases: []string{"d"},
			Usage:   "enable debug output",
		},
	}
}

func buildCommands() []*cli.Command {
	return []*cli.Command{
		{
			Name:    "fish",
			Aliases: []string{"f"},
			Action:  fish,
			Usage:   "generate the fish shell completion",
		},
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	if cmd.Bool(debugFlag) {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("Enabled debug output")
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	config := modiff.NewConfig(
		cmd.String(repositoryArg),
		cmd.String(fromArg),
		cmd.String(toArg),
		cmd.Bool(linkArg),
		cmd.Uint(headerLevelArg),
	).
		WithFormat(cmd.String(formatArg)).
		WithFilter(cmd.String(filterArg)).
		WithConcurrency(cmd.Uint(concurrencyArg))

	result, err := modiff.Run(ctx, config)
	if err != nil {
		return fmt.Errorf("unable to run: %w", err)
	}

	logrus.Info("Done, the result will be printed to `stdout`")

	_, err = os.Stdout.WriteString(result)
	if err != nil {
		return fmt.Errorf("unable to write result: %w", err)
	}

	return nil
}

func fish(_ context.Context, cmd *cli.Command) error {
	result, err := cmd.Root().ToFishCompletion()
	if err != nil {
		return fmt.Errorf("unable to generate completions: %w", err)
	}

	_, err = os.Stdout.WriteString(result)
	if err != nil {
		return fmt.Errorf("unable to write completions: %w", err)
	}

	return nil
}
