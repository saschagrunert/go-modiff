package writer

import (
	"github.com/urfave/cli"
)

type Cli struct {
	App          *cli.App
	Date         string
	Commands     []string
	GlobalArgs   []string
	SynopsisArgs []string
}

const cliTemplateString = `% {{ .App.Name }}(8) {{ .App.Description }}
% {{ .App.Author }}
% {{ .Date }}

# NAME

{{ .App.Name }} - {{ .App.Usage }}

# SYNOPSIS

{{ .App.Name }}

` + "```" + `{{ range $v := .SynopsisArgs }}
{{ $v }}{{ end }}
` + "```" + `

# DESCRIPTION

{{ .App.UsageText }}

**Usage**:

` + "```" + `
{{ .App.Name }} [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
` + "```" + `

# GLOBAL OPTIONS
{{ range $v := .GlobalArgs }}
{{ $v }}{{ end }}

# COMMANDS
{{ range $v := .Commands }}
{{ $v }}{{ end }}`
