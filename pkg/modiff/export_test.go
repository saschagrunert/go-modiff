package modiff

//nolint:gochecknoglobals // exported for testing
var (
	GoProxyURLForTest      = goProxyURL
	ParseModuleLineForTest = parseModuleLine
	PrettifyVersionForTest = prettifyVersion
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
