package writer

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/urfave/cli"
)

func New(app *cli.App) *Cli {
	now := time.Now()
	return &Cli{
		App:          app,
		Date:         fmt.Sprintf("%s %d", now.Month(), now.Year()),
		Commands:     prepareCommands(app.Commands, 0),
		GlobalArgs:   prepareArgsWithValues(app.Flags),
		SynopsisArgs: prepareArgsSynopsis(app.Flags),
	}
}

func (c *Cli) Write(w io.Writer) error {
	const name = "cli"
	t, err := template.New(name).Parse(cliTemplateString)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, name, c)
}

const nl = "\n"
const noDescription = "_no description available_"

func prepareCommands(commands []cli.Command, level int) []string {
	coms := []string{}
	for i := range commands {
		command := &commands[i]
		prepared := strings.Repeat("#", level+2) + " " +
			strings.Join(command.Names(), ", ") + nl

		usage := noDescription
		if command.Usage != "" {
			usage = command.Usage
		}
		prepared += nl + usage + nl

		flags := prepareArgsWithValues(command.Flags)
		if len(flags) > 0 {
			prepared += nl
		}
		prepared += strings.Join(flags, nl)
		if len(flags) > 0 {
			prepared += nl
		}

		coms = append(coms, prepared)

		// recursevly iterate subcommands
		if len(command.Subcommands) > 0 {
			coms = append(
				coms,
				prepareCommands(command.Subcommands, level+1)...,
			)
		}
	}

	return coms
}

func prepareArgsWithValues(flags []cli.Flag) []string {
	return prepareFlags(flags, ", ", "**", "**", `""`, true)
}

func prepareArgsSynopsis(flags []cli.Flag) []string {
	return prepareFlags(flags, "|", "[", "]", "[value]", false)
}

func prepareFlags(
	flags []cli.Flag,
	sep, opener, closer, value string,
	addDetails bool,
) []string {
	args := []string{}
	for _, flag := range flags {
		modifiedArg := opener
		for _, s := range strings.Split(flag.GetName(), ",") {
			trimmed := strings.TrimSpace(s)
			if len(modifiedArg) > len(opener) {
				modifiedArg += sep
			}
			if len(trimmed) > 1 {
				modifiedArg += "--" + trimmed
			} else {
				modifiedArg += "-" + trimmed
			}
		}
		modifiedArg += closer
		if flagTakesValue(flag) {
			modifiedArg += "=" + value
		}

		if addDetails {
			modifiedArg += flagDetails(flag)
		}

		args = append(args, modifiedArg)

	}
	sort.Strings(args)
	return args
}

// flagTakesValue returns true if the flag takes a value, otherwise false
func flagTakesValue(flag cli.Flag) bool {
	if _, ok := flag.(cli.BoolFlag); ok {
		return false
	}
	if _, ok := flag.(cli.BoolTFlag); ok {
		return false
	}
	if _, ok := flag.(cli.DurationFlag); ok {
		return true
	}
	if _, ok := flag.(cli.Float64Flag); ok {
		return true
	}
	if _, ok := flag.(cli.GenericFlag); ok {
		return true
	}
	if _, ok := flag.(cli.Int64Flag); ok {
		return true
	}
	if _, ok := flag.(cli.IntFlag); ok {
		return true
	}
	if _, ok := flag.(cli.IntSliceFlag); ok {
		return true
	}
	if _, ok := flag.(cli.Int64SliceFlag); ok {
		return true
	}
	if _, ok := flag.(cli.StringFlag); ok {
		return true
	}
	if _, ok := flag.(cli.StringSliceFlag); ok {
		return true
	}
	if _, ok := flag.(cli.Uint64Flag); ok {
		return true
	}
	if _, ok := flag.(cli.UintFlag); ok {
		return true
	}
	return false
}

// flagDetails returns a string containing the flags metadata
func flagDetails(flag cli.Flag) string {
	description := ""
	value := ""
	if f, ok := flag.(cli.BoolFlag); ok {
		description = f.Usage
	}
	if f, ok := flag.(cli.BoolTFlag); ok {
		description = f.Usage
	}
	if f, ok := flag.(cli.DurationFlag); ok {
		description = f.Usage
		value = f.Value.String()
	}
	if f, ok := flag.(cli.Float64Flag); ok {
		description = f.Usage
		value = fmt.Sprintf("%f", f.Value)
	}
	if f, ok := flag.(cli.GenericFlag); ok {
		description = f.Usage
		if f.Value != nil {
			value = f.Value.String()
		}
	}
	if f, ok := flag.(cli.Int64Flag); ok {
		description = f.Usage
		value = fmt.Sprintf("%d", f.Value)
	}
	if f, ok := flag.(cli.IntFlag); ok {
		description = f.Usage
		value = fmt.Sprintf("%d", f.Value)
	}
	if f, ok := flag.(cli.IntSliceFlag); ok {
		description = f.Usage
		if f.Value != nil {
			value = f.Value.String()
		}
	}
	if f, ok := flag.(cli.Int64SliceFlag); ok {
		description = f.Usage
		if f.Value != nil {
			value = f.Value.String()
		}
	}
	if f, ok := flag.(cli.StringFlag); ok {
		description = f.Usage
		value = f.Value
	}
	if f, ok := flag.(cli.StringSliceFlag); ok {
		description = f.Usage
		if f.Value != nil {
			value = f.Value.String()
		}
	}
	if f, ok := flag.(cli.Uint64Flag); ok {
		description = f.Usage
		value = fmt.Sprintf("%d", f.Value)
	}
	if f, ok := flag.(cli.UintFlag); ok {
		description = f.Usage
		value = fmt.Sprintf("%d", f.Value)
	}
	if description == "" {
		description = noDescription
	}
	if value != "" {
		description += " (default: " + value + ")"
	}
	return ": " + description
}
