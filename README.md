# go-modiff 📔

[![ci](https://github.com/saschagrunert/go-modiff/actions/workflows/ci.yml/badge.svg)](https://github.com/saschagrunert/go-modiff/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/saschagrunert/go-modiff/branch/main/graph/badge.svg)](https://codecov.io/gh/saschagrunert/go-modiff)

## Command line tool for diffing go module dependency changes between versions

## Usage

The tool can be installed via:

```shell
go install github.com/saschagrunert/go-modiff/cmd/go-modiff@latest
```

After that, the application can be used like this:

```shell
> go-modiff -r github.com/cri-o/cri-o -f v1.35.0
INFO Cloning reference repository github.com/cri-o/cri-o
INFO Setting up 'from' at v1.35.0
INFO Setting up 'to' at HEAD
INFO Processing module diffs
INFO 385 modules found
INFO 1 modules added
INFO 11 modules changed
INFO 0 modules removed
INFO Done, the result will be printed to `stdout`
```

```markdown
# Dependencies

## Added

- github.com/creack/pty: v1.1.7

## Changed

- github.com/containerd/go-runc: 7d11b49 → 9007c24
- github.com/containerd/project: 831961d → 7fb81da
- github.com/containerd/ttrpc: 2a805f7 → 1fb3814
- github.com/containers/libpod: 5e42bf0 → v1.4.4
- github.com/containers/storage: v1.12.12 → v1.12.13
- github.com/godbus/dbus: 2ff6f7f → 8a16820
- github.com/kr/pty: v1.1.5 → v1.1.8
- golang.org/x/net: 3b0461e → da137c7
- golang.org/x/sys: c5567b4 → 04f50cd
- google.golang.org/grpc: v1.21.1 → v1.22.0
- honnef.co/go/tools: e561f67 → ea95bdf

## Removed

_Nothing has changed._
```

It is also possible to add diff links to the markdown output via `--link, -l`.
The output would then look like this:

```markdown
# Dependencies

## Added

- github.com/shurcooL/httpfs: [8d4bc4b](https://github.com/shurcooL/httpfs/tree/8d4bc4b)
- github.com/shurcooL/vfsgen: [6a9ea43](https://github.com/shurcooL/vfsgen/tree/6a9ea43)

## Changed

- github.com/onsi/ginkgo: [v1.8.0 → v1.9.0](https://github.com/onsi/ginkgo/compare/v1.8.0...v1.9.0)
- github.com/onsi/gomega: [v1.5.0 → v1.6.0](https://github.com/onsi/gomega/compare/v1.5.0...v1.6.0)
- github.com/saschagrunert/ccli: [e981d95 → 05e6f25](https://github.com/saschagrunert/ccli/compare/e981d95...05e6f25)
- github.com/urfave/cli: [v1.20.0 → 23c8303](https://github.com/urfave/cli/compare/v1.20.0...23c8303)

## Removed

- github.com/saschagrunert/go-docgen: [v0.1.3](https://github.com/saschagrunert/go-docgen/tree/v0.1.3)
```

### JSON output

Use `--format json` (or `-o json`) for machine-readable output:

```json
{
  "added": [],
  "changed": [
    {
      "name": "github.com/onsi/ginkgo",
      "before": "v1.8.0",
      "after": "v1.9.0"
    }
  ],
  "removed": []
}
```

### Filtering

Use `--filter` to show only a specific category:

```shell
go-modiff -r github.com/owner/repo -f v1.0.0 -t v2.0.0 --filter added
```

### Arguments

The following command line arguments are currently supported:

| Argument              | Description                                                         |
| --------------------- | ------------------------------------------------------------------- |
| `--repository, -r`    | repository to be used, like: github.com/owner/repo                  |
| `--from, -f`          | the start of the comparison (any valid git rev) (default: "HEAD") |
| `--to, -t`            | the end of the comparison (any valid git rev) (default: "HEAD")   |
| `--link, -l`          | add diff links to the markdown output (default: false)              |
| `--header-level, -i`  | markdown header level depth (default: 1)                            |
| `--format, -o`        | output format: markdown or json (default: "markdown")               |
| `--filter`            | filter by category: added, changed, or removed                     |
| `--concurrency, -c`   | concurrent proxy requests for link resolution (default: 10)         |
| `--debug, -d`         | enable debug output (default: false)                                |

### How links work

When `--link` is enabled, go-modiff queries the Go module proxy
(`proxy.golang.org`) to resolve VCS metadata (repository URL, commit hash, git
ref) for each changed module. This provides accurate commit and compare links
for GitHub and `go.googlesource.com` hosted modules. For modules where the proxy
does not return origin data, go-modiff falls back to URL-pattern-based link
generation for GitHub modules.

The `--concurrency` flag controls how many proxy requests run in parallel
(default: 10). Increase it for faster link resolution on large diffs, or
decrease it to reduce load on the proxy.

## Contributing

You want to contribute to this project? Wow, thanks! So please just fork it and
send me a pull request.
