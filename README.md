# go-modiff 📔

[![CircleCI](https://circleci.com/gh/saschagrunert/go-modiff.svg?style=shield)](https://circleci.com/gh/saschagrunert/go-modiff)

## Command line tool for diffing go module dependency changes between versions

## Usage

The tool can be installed via:

```shell
go get github.com/saschagrunert/go-modiff/cmd/go-modiff
```

After that, the application can be used like this:

```shell
> go-modiff -r github.com/cri-o/cri-o -f v1.15.0
INFO Cloning github.com/cri-o/cri-o into /tmp/go-modiff844484466
INFO Retrieving modules of v1.15.0
INFO Retrieving modules of master
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

The following command line arguments are currently supported:

| Argument           | Description                                                         |
| ------------------ | ------------------------------------------------------------------- |
| `--repository, -r` | repository to be used, like: github.com/owner/repo                  |
| `--from, -f`       | the start of the comparison (any valid git rev) (default: "master") |
| `--to, -t`         | the end of the comparison (any valid git ref) (default: "master")   |

## Contributing

You want to contribute to this project? Wow, thanks! So please just fork it and
send me a pull request.
