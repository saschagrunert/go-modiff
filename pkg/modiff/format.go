package modiff

import (
	"encoding/json"
	"fmt"
	"strings"
)

func formatModuleMarkdown(change ModuleChange, category string, addLinks bool) string {
	txt := fmt.Sprintf("- %s: ", change.Name)

	switch category {
	case FilterAdded:
		if addLinks && change.Link != "" {
			txt += fmt.Sprintf("[%s](%s)", change.After, change.Link)
		} else {
			txt += change.After
		}
	case FilterRemoved:
		if addLinks && change.Link != "" {
			txt += fmt.Sprintf("[%s](%s)", change.Before, change.Link)
		} else {
			txt += change.Before
		}
	case FilterChanged:
		if addLinks && change.Link != "" {
			txt += fmt.Sprintf("[%s → %s](%s)", change.Before, change.After, change.Link)
		} else {
			txt += fmt.Sprintf("%s → %s", change.Before, change.After)
		}
	}

	return txt
}

func formatMarkdown(result DiffResult, addLinks bool, headerLevel uint, filter string) string {
	level := min(headerLevel, maxHeaderLevel)
	builder := &strings.Builder{}

	fmt.Fprintf(
		builder, "%s Dependencies\n", strings.Repeat("#", int(level)),
	)

	writeSection := func(section string, changes []ModuleChange, category string) {
		fmt.Fprintf(
			builder,
			"\n%s %s\n", strings.Repeat("#", min(int(level)+1, maxHeaderLevel)), section,
		)

		if len(changes) > 0 {
			for _, change := range changes {
				fmt.Fprintf(builder, "%s\n", formatModuleMarkdown(change, category, addLinks))
			}
		} else {
			builder.WriteString("_Nothing has changed._\n")
		}
	}

	if filter == "" || filter == FilterAdded {
		writeSection("Added", result.Added, FilterAdded)
	}

	if filter == "" || filter == FilterChanged {
		writeSection("Changed", result.Changed, FilterChanged)
	}

	if filter == "" || filter == FilterRemoved {
		writeSection("Removed", result.Removed, FilterRemoved)
	}

	return builder.String()
}

func formatJSON(result DiffResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling JSON result: %w", err)
	}

	return string(data) + "\n", nil
}
