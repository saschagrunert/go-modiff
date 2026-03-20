package modiff

//nolint:gochecknoglobals // exported for testing
var (
	GoProxyURLForTest      = goProxyURL
	ParseModuleLineForTest = parseModuleLine
	PrettifyVersionForTest = prettifyVersion
	GitHubBaseURLForTest   = gitHubBaseURL
)

// NewGoModInfoForTest creates a goModInfo for testing.
func NewGoModInfoForTest(vcsURL, hash, ref string) *goModInfo {
	return &goModInfo{
		Version: "",
		Origin: goModOrigin{
			VCS:  "git",
			URL:  vcsURL,
			Hash: hash,
			Ref:  ref,
		},
	}
}

// NewEntryForTest creates an entry for testing.
func NewEntryForTest(before, after string) entry {
	return entry{beforeVersion: before, afterVersion: after}
}

// RefNameForTest calls refName on a goModInfo.
func RefNameForTest(info *goModInfo) string {
	return info.refName()
}

// CommitURLForTest calls commitURL on a goModInfo.
func CommitURLForTest(info *goModInfo) string {
	return info.commitURL()
}

// CompareURLForTest calls compareURL on two goModInfos.
func CompareURLForTest(info, other *goModInfo) string {
	return info.compareURL(other)
}

// ClassifyModuleForTest calls classifyModule.
func ClassifyModuleForTest(
	mod entry, name string, beforeInfo, afterInfo *goModInfo, addLinks bool,
) (string, ModuleChange) {
	return classifyModule(mod, name, beforeInfo, afterInfo, addLinks)
}

// GenerateSingleLinkForTest calls generateSingleLink.
func GenerateSingleLinkForTest(display string, info *goModInfo, name string) string {
	return generateSingleLink(display, info, name)
}

// ApplyFilterForTest calls applyFilter.
func ApplyFilterForTest(result *DiffResult, filter string) {
	applyFilter(result, filter)
}
