package update

import (
	"fmt"
	"strings"
)

// semanticVersion implements the ordering rules needed for published tags.
// The numeric fields stay as validated strings so comparison cannot overflow.
type semanticVersion struct {
	major      string
	minor      string
	patch      string
	prerelease []string
}

func parseSemanticVersion(tag string) (semanticVersion, error) {
	if !strings.HasPrefix(tag, "v") {
		return semanticVersion{}, fmt.Errorf("must start with v")
	}
	mainAndPre, _, hasBuild := strings.Cut(strings.TrimPrefix(tag, "v"), "+")
	if mainAndPre == "" {
		return semanticVersion{}, fmt.Errorf("missing version")
	}
	main, prerelease, hasPrerelease := strings.Cut(mainAndPre, "-")
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return semanticVersion{}, fmt.Errorf("must contain major.minor.patch")
	}
	major, err := parseSemanticNumber(parts[0])
	if err != nil {
		return semanticVersion{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := parseSemanticNumber(parts[1])
	if err != nil {
		return semanticVersion{}, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := parseSemanticNumber(parts[2])
	if err != nil {
		return semanticVersion{}, fmt.Errorf("invalid patch version: %w", err)
	}
	parsed := semanticVersion{major: major, minor: minor, patch: patch}
	if hasPrerelease {
		if !validPrerelease(prerelease) {
			return semanticVersion{}, fmt.Errorf("invalid prerelease %q", prerelease)
		}
		parsed.prerelease = strings.Split(prerelease, ".")
	}
	// Build metadata does not affect precedence. Its grammar is still checked so
	// malformed tags cannot become update candidates.
	if hasBuild {
		build := strings.TrimPrefix(tag, "v")
		_, build, _ = strings.Cut(build, "+")
		if !validIdentifiers(build, false) {
			return semanticVersion{}, fmt.Errorf("invalid build metadata %q", build)
		}
	}
	return parsed, nil
}

func parseSemanticNumber(value string) (string, error) {
	if !isNumeric(value) || (len(value) > 1 && value[0] == '0') {
		return "", fmt.Errorf("%q is not a canonical numeric identifier", value)
	}
	return value, nil
}

func validPrerelease(value string) bool {
	return validIdentifiers(value, true)
}

func validIdentifiers(value string, rejectLeadingZeroNumbers bool) bool {
	if value == "" {
		return false
	}
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		for _, character := range identifier {
			if !((character >= '0' && character <= '9') || (character >= 'A' && character <= 'Z') || (character >= 'a' && character <= 'z') || character == '-') {
				return false
			}
		}
		if rejectLeadingZeroNumbers && isNumeric(identifier) && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

func (v semanticVersion) isPrerelease() bool {
	return len(v.prerelease) != 0
}

func (v semanticVersion) compare(other semanticVersion) int {
	for _, pair := range [][2]string{{v.major, other.major}, {v.minor, other.minor}, {v.patch, other.patch}} {
		if compared := compareNumber(pair[0], pair[1]); compared != 0 {
			return compared
		}
	}
	if !v.isPrerelease() && !other.isPrerelease() {
		return 0
	}
	if !v.isPrerelease() {
		return 1
	}
	if !other.isPrerelease() {
		return -1
	}
	for index := 0; index < len(v.prerelease) && index < len(other.prerelease); index++ {
		if compared := compareIdentifier(v.prerelease[index], other.prerelease[index]); compared != 0 {
			return compared
		}
	}
	if len(v.prerelease) < len(other.prerelease) {
		return -1
	}
	if len(v.prerelease) > len(other.prerelease) {
		return 1
	}
	return 0
}

func compareIdentifier(first, second string) int {
	if isNumeric(first) && isNumeric(second) {
		return compareNumber(first, second)
	}
	if isNumeric(first) {
		return -1
	}
	if isNumeric(second) {
		return 1
	}
	return strings.Compare(first, second)
}

func compareNumber(first, second string) int {
	if len(first) < len(second) {
		return -1
	}
	if len(first) > len(second) {
		return 1
	}
	return strings.Compare(first, second)
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
