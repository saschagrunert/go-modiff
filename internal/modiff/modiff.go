package modiff

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

type versions struct {
	before string
	after  string
}

type modules = map[string]versions

const (
	RepositoryArg = "repository"
	FromArg       = "from"
	ToArg         = "to"
	LinkArg       = "link"
)

func Run(c *cli.Context) (string, error) {
	// Enable to modules
	os.Setenv("GO111MODULE", "on")

	if c == nil {
		return logErr("cli context is nil")
	}
	// Validate the flags
	if c.String(RepositoryArg) == "" {
		return logErr("argument %q is required", RepositoryArg)
	}
	repository := c.String(RepositoryArg)
	from := c.String(FromArg)
	to := c.String(ToArg)
	if from == to {
		return logErr("no diff possible if `--from` equals `--to`")
	}

	// Prepare the environment
	dir, err := ioutil.TempDir("", "go-modiff")
	if err != nil {
		return logErr(err.Error())
	}
	defer os.RemoveAll(dir)

	logrus.Infof("Setting up repository %s", repository)

	if _, err := execCmd(dir, "git init"); err != nil {
		return logErr(err.Error())
	}

	if _, err := execCmd(
		dir, "git remote add origin %s", toURL(c.String(RepositoryArg)),
	); err != nil {
		return logErr(err.Error())
	}

	// Retrieve and diff the modules
	mods, err := getModules(dir, from, to)
	if err != nil {
		return "", err
	}
	return diffModules(mods, c.Bool(LinkArg)), nil
}

func toURL(name string) string {
	return "https://" + name
}

func isGitHubURL(name string) bool {
	return strings.HasPrefix(name, "github.com")
}

func sanitizeTag(tag string) string {
	return strings.TrimSuffix(tag, "+incompatible")
}

func logErr(format string, a ...interface{}) (string, error) {
	err := fmt.Errorf(format, a...)
	logrus.Error(err)
	return "", err
}

func diffModules(mods modules, addLinks bool) string {
	var added, removed, changed []string
	for name, mod := range mods {
		txt := fmt.Sprintf("- %s: ", name)
		if mod.before == "" { // nolint: gocritic
			if addLinks && isGitHubURL(name) {
				txt += fmt.Sprintf("[%s](%s/tree/%s)",
					mod.after, toURL(name), sanitizeTag(mod.after))
			} else {
				txt += mod.after
			}
			added = append(added, txt)
		} else if mod.after == "" {
			if addLinks && isGitHubURL(name) {
				txt += fmt.Sprintf("[%s](%s/tree/%s)",
					mod.before, toURL(name), sanitizeTag(mod.before))
			} else {
				txt += mod.before
			}
			removed = append(removed, txt)
		} else if mod.before != mod.after {
			if addLinks && isGitHubURL(name) {
				txt += fmt.Sprintf("[%s → %s](%s/compare/%s...%s)",
					mod.before, mod.after, toURL(name),
					sanitizeTag(mod.before), sanitizeTag(mod.after))
			} else {
				txt += fmt.Sprintf("%s → %s", mod.before, mod.after)
			}
			changed = append(changed, txt)
		}
	}
	sort.Strings(added)
	sort.Strings(changed)
	sort.Strings(removed)
	logrus.Infof("%d modules added", len(added))
	logrus.Infof("%d modules changed", len(changed))
	logrus.Infof("%d modules removed", len(removed))

	// Pretty print
	builder := &strings.Builder{}
	builder.WriteString("# Dependencies\n")
	forEach := func(section string, input []string) {
		builder.WriteString(fmt.Sprintf("\n## %s\n", section))
		if len(input) > 0 {
			for _, mod := range input {
				builder.WriteString(fmt.Sprintf("%s\n", mod))
			}
		} else {
			builder.WriteString("_Nothing has changed._\n")
		}
	}
	forEach("Added", added)
	forEach("Changed", changed)
	forEach("Removed", removed)
	return builder.String()
}

func getModules(workDir, from, to string) (modules, error) {
	// Retrieve all modules
	before, err := retrieveModules(from, workDir)
	if err != nil {
		return nil, err
	}
	after, err := retrieveModules(to, workDir)
	if err != nil {
		return nil, err
	}

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
	return res, nil
}

func retrieveModules(rev, workDir string) (string, error) {
	logrus.Infof("Retrieving modules of %s", rev)
	_, err := execCmd(workDir, "git fetch --depth=1 origin %s", rev)
	if err != nil {
		logrus.Error(err)
		return "", err
	}

	_, err = execCmd(workDir, "git checkout -f FETCH_HEAD")
	if err != nil {
		logrus.Error(err)
		return "", err
	}

	mods, err := execCmd(workDir, "go list -m all")
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	return mods, nil
}

func execCmd(workDir, format string, args ...interface{}) (string, error) {
	var (
		cmd            *exec.Cmd
		stdout, stderr bytes.Buffer
	)
	command := fmt.Sprintf(format, args...)
	c := strings.Split(command, " ")
	cmd = exec.Command(c[0], c[1:]...)
	cmd.Dir = workDir
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

	output := stdout.String()
	if output != "" {
		logrus.Debug(output)
	}
	return output, nil
}
