package main

import (
	"fmt"
	"os"

	"github.com/saschagrunert/ccli"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/saschagrunert/go-modiff/internal/modiff"
)

const (
	repositoryArg = "repository"
	fromArg       = "from"
	toArg         = "to"
	linkArg       = "link"
)

func main() {
	const debugFlag = "debug"

	app := ccli.NewApp()
	app.Name = "go-modiff"
	app.Version = "1.0.0"
	app.Authors = []*cli.Author{
		{Name: "Sascha Grunert", Email: "mail@saschagrunert.de"},
	}
	app.Usage = "Command line tool for diffing go module " +
		"dependency changes between versions"
	app.UsageText = app.Usage
	app.UseShortOptionHandling = true
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:      repositoryArg,
			Aliases:   []string{"r"},
			Usage:     "repository to be used, like: github.com/owner/repo",
			TakesFile: false,
		},
		&cli.StringFlag{
			Name:      fromArg,
			Aliases:   []string{"f"},
			Value:     "master",
			Usage:     "the start of the comparison, any valid git rev",
			TakesFile: false,
		},
		&cli.StringFlag{
			Name:      toArg,
			Aliases:   []string{"t"},
			Value:     "master",
			Usage:     "the end of the comparison, any valid git rev",
			TakesFile: false,
		},
		&cli.BoolFlag{
			Name:    linkArg,
			Aliases: []string{"l"},
			Usage:   "add diff links to the markdown output",
		},
		&cli.BoolFlag{
			Name:    debugFlag,
			Aliases: []string{"d"},
			Usage:   "enable debug output",
		},
	}
	app.Commands = []*cli.Command{{
		Name:    "docs",
		Aliases: []string{"d"},
		Action:  docs,
		Usage: "generate the markdown or man page documentation " +
			"and print it to stdout",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "markdown",
				Usage: "print the markdown version",
			},
			&cli.BoolFlag{
				Name:  "man",
				Usage: "print the man version",
			},
		},
	}, {
		Name:    "fish",
		Aliases: []string{"f"},
		Action:  fish,
		Usage:   "generate the fish shell completion",
	}}
	app.Action = func(c *cli.Context) error {
		// Init the logging facade
		logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
		if c.Bool("debug") {
			logrus.SetLevel(logrus.DebugLevel)
			logrus.Debug("Enabled debug output")
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}

		// Run modiff
		config := modiff.NewConfig(
			c.String(repositoryArg),
			c.String(fromArg),
			c.String(toArg),
			c.Bool(linkArg),
		)
		res, err := modiff.Run(config)
		if err != nil {
			return err
		}
		logrus.Info("Done, the result will be printed to `stdout`")
		fmt.Print(res)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func docs(c *cli.Context) (err error) {
	res := ""
	if c.Bool("markdown") {
		res, err = c.App.ToMarkdown()
	} else if c.Bool("man") {
		res, err = c.App.ToMan()
	}
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", res)
	return nil
}

func fish(c *cli.Context) (err error) {
	res, err := c.App.ToFishCompletion()
	if err != nil {
		return err
	}
	fmt.Printf("%v", res)
	return nil
}
