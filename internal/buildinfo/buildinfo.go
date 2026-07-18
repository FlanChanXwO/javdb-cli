// Package buildinfo exposes metadata embedded by the Go linker at build time.
package buildinfo

var (
	// Overridden via -ldflags -X at release build time.
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// Info is safe-to-print build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// Current returns embedded build metadata.
// Version is normalized to include a leading "v" when it looks like a SemVer
// without one, so brew tests can assert `version_info["version"] == "v#{version}"`.
func Current() Info {
	return Info{
		Version:   NormalizeVersion(Version),
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

// NormalizeVersion ensures release versions are "vX.Y.Z".
// Leaves "dev" and already-prefixed values unchanged.
func NormalizeVersion(v string) string {
	if v == "" || v == "dev" {
		if v == "" {
			return "dev"
		}
		return v
	}
	if len(v) > 0 && v[0] == 'v' {
		return v
	}
	// digit-leading SemVer
	if v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}

// IsDevelopment reports whether this is a local/dev build.
func (info Info) IsDevelopment() bool {
	return info.Version == "dev"
}
