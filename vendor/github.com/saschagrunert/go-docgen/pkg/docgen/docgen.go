package docgen

import (
	"bytes"
	"io"

	"github.com/saschagrunert/go-docgen/internal/writer"

	"github.com/cpuguy83/go-md2man/md2man"
	"github.com/urfave/cli"
)

// CliToMarkdown converts a given `cli.App` to a markdown string.
// The function errors if either parsing or writing of the string fails.
func CliToMarkdown(app *cli.App) (string, error) {
	var w bytes.Buffer
	if err := write(app, &w); err != nil {
		return "", err
	}
	return w.String(), nil
}

// CliToMan converts a given `cli.App` to a man page string.
// The function errors if either parsing or writing of the string fails.
func CliToMan(app *cli.App) (string, error) {
	var w bytes.Buffer
	if err := write(app, &w); err != nil {
		return "", err
	}
	man := md2man.Render(w.Bytes())
	return string(man), nil
}

func write(app *cli.App, w io.Writer) error {
	return writer.New(app).Write(w)
}
