% go-modiff(8) 

% Sascha Grunert

# NAME

go-modiff - Command line tool for diffing go module dependency changes between versions

# SYNOPSIS

go-modiff

```
[--debug]
[--from]=[value]
[--help|-h]
[--link]
[--repository]=[value]
[--to]=[value]
[--version|-v]
```

# DESCRIPTION

Command line tool for diffing go module dependency changes between versions

**Usage**:

```
go-modiff [GLOBAL OPTIONS] command [COMMAND OPTIONS] [ARGUMENTS...]
```

# GLOBAL OPTIONS

**--debug**: enable debug output

**--from**="": the start of the comparison, any valid git rev (default: master)

**--help, -h**: show help

**--link**: add diff links to the markdown output

**--repository**="": repository to be used, like: github.com/owner/repo

**--to**="": the end of the comparison, any valid git rev (default: master)

**--version, -v**: print the version


# COMMANDS

## docs, d

generate the markdown or man page documentation and print it to stdout

**--help, -h**: show help

**--man**: print the man version

**--markdown**: print the markdown version

## fish, f

generate the fish shell completion

## help, h

Shows a list of commands or help for one command

