package process

// ExecutableName returns the platform-specific javdb executable name.
func ExecutableName(goos string) string {
	if goos == "windows" {
		return "javdb.exe"
	}
	return "javdb"
}
