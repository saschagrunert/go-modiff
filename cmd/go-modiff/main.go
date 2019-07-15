package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	repositoryArg = "repository"
	fromArg       = "from"
	toArg         = "to"
)

type versions struct {
	before string
	after  string
}

type modules = map[string]versions

func main() {
	// Init the logging facade
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	logrus.SetLevel(logrus.DebugLevel)

	// Enable to modules
	os.Setenv("GO111MODULE", "on")

	app := cli.NewApp()
	app.Name = "go-modiff"
	app.Version = "0.2.0-dev"
	app.Author = "Sascha Grunert"
	app.Email = "mail@saschagrunert.de"
	app.Usage = "Command line tool for diffing go module " +
		"dependency changes between versions"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  repositoryArg + ", r",
			Usage: "repository to be used, like: github.com/owner/repo",
		},
		cli.StringFlag{
			Name:  fromArg + ", f",
			Value: "master",
			Usage: "the start of the comparison (any valid git rev)",
		},
		cli.StringFlag{
			Name:  toArg + ", t",
			Value: "master",
			Usage: "the end of the comparison (any valid git ref)",
		},
	}
	app.Action = run
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	// Validate the flags
	if c.String(repositoryArg) == "" {
		logrus.Fatalf("Argument %q is required", repositoryArg)
	}
	repository := c.String(repositoryArg)
	from := c.String(fromArg)
	to := c.String(toArg)
	if from == to {
		logrus.Fatal("No diff possible if `--from` equals `--to`")
	}

	// Prepare the environment
	dir, err := ioutil.TempDir("", "go-modiff")
	if err != nil {
		logrus.Fatal(err)
	}
	defer os.RemoveAll(dir)
	logrus.Infof("Cloning %s into %s", repository, dir)
	if _, err := execCmd(
		fmt.Sprintf("git clone https://%s %s", c.String(repositoryArg), dir),
		os.TempDir(),
	); err != nil {
		logrus.Fatal(err)
	}

	// Retrieve and diff the modules
	mods := getModules(dir, from, to)
	diffModules(mods)
	return nil
}

func diffModules(mods modules) {
	var added, removed, changed []string
	for name, mod := range mods {
		// nolint: gocritic
		if mod.before == "" {
			added = append(
				added,
				fmt.Sprintf("- %s: %s", name, mod.after),
			)
		} else if mod.after == "" {
			removed = append(
				removed,
				fmt.Sprintf("- %s: %s", name, mod.before),
			)
		} else if mod.before != mod.after {
			changed = append(
				changed,
				fmt.Sprintf("- %s: %s → %s", name, mod.before, mod.after),
			)
		}
	}
	sort.Strings(added)
	sort.Strings(changed)
	sort.Strings(removed)
	logrus.Infof("%d modules added", len(added))
	logrus.Infof("%d modules changed", len(changed))
	logrus.Infof("%d modules removed", len(removed))

	// Pretty print
	logrus.Infof("Done, the result will be printed to `stdout`")
	fmt.Printf("\n# Dependencies\n")
	forEach := func(section string, input []string) {
		fmt.Printf("\n## %s\n", section)
		if len(input) > 0 {
			for _, mod := range input {
				fmt.Println(mod)
			}
		} else {
			fmt.Println("_Nothing has changed._")
		}
	}
	forEach("Added", added)
	forEach("Changed", changed)
	forEach("Removed", removed)
}

func getModules(workDir, from, to string) modules {
	// Retrieve all modules
	before := retrieveModules(from, workDir)
	after := retrieveModules(to, workDir)

	// Parse the modules
	res := modules{}

	forEach := func(input string, do func(res *versions, version string)) {
		scanner := bufio.NewScanner(strings.NewReader(input))
		for scanner.Scan() {
			// Skip version-less modules, like the local one
			split := strings.Split(scanner.Text(), " ")
			if len(split) < 2 {
				continue
			}
			// Rewrites have to be handled differently
			if len(split) > 2 && split[2] == "=>" {
				// Local rewrites without any version will be skipped
				if len(split) == 4 {
					continue
				}

				// Use the rewritten version and name if available
				if len(split) == 5 {
					split[0] = split[3]
					split[1] = split[4]
				}

			}
			name := strings.TrimSpace(split[0])
			version := strings.TrimSpace(split[1])

			// Prettify pseudo versions
			vSplit := strings.Split(version, "-")
			if len(vSplit) > 2 {
				v := vSplit[len(vSplit)-1]
				if len(v) > 7 {
					version = v[:7]
				} else {
					// This should never happen but who knows what go modules
					// will do next
					version = v
				}
			}

			// Process the entry
			entry := &versions{}
			if val, ok := res[name]; ok {
				entry = &val
			}
			do(entry, version)
			res[name] = *entry
		}
	}
	forEach(before, func(res *versions, v string) { res.before = v })
	forEach(after, func(res *versions, v string) { res.after = v })

	logrus.Infof("%d modules found", len(res))
	return res
}

func retrieveModules(ref, workDir string) string {
	logrus.Infof("Retrieving modules of %s", ref)
	_, err := execCmd("git checkout -f "+ref, workDir)
	if err != nil {
		logrus.Fatal(err)
	}

	mods, err := execCmd("go list -m all", workDir)
	if err != nil {
		logrus.Fatal(err)
	}
	return mods
}

func execCmd(command, workDir string) (string, error) {
	c := strings.Split(command, " ")

	var cmd *exec.Cmd
	if len(c) == 0 {
		cmd = exec.Command(c[0])
	} else {
		cmd = exec.Command(c[0], c[1:]...)
	}
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		return "", fmt.Errorf(
			"`%v` failed: %v %v (%v)",
			command,
			stderr.String(),
			stdout.String(),
			err,
		)
	}

	return stdout.String(), nil
}
