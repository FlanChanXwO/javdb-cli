// Command platformmatrix validates the shared CI platform registry and emits
// GitHub Actions matrix JSON for one capability.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type registry struct {
	Platforms []platform `json:"platforms"`
}

type platform struct {
	GOOS         string   `json:"goos"`
	GOARCH       string   `json:"goarch"`
	Runner       string   `json:"runner"`
	Capabilities []string `json:"capabilities"`
}

type matrixEntry struct {
	GOOS         string   `json:"goos"`
	GOARCH       string   `json:"goarch"`
	Runner       string   `json:"runner"`
	Capabilities []string `json:"capabilities"`
	Artifact     string   `json:"artifact"`
}

type matrix struct {
	Include []matrixEntry `json:"include"`
}

var allowedCapabilities = map[string]struct{}{
	"container":    {},
	"homebrew":     {},
	"race":         {},
	"release":      {},
	"smoke":        {},
	"verification": {},
}

func main() {
	registryPath := flag.String("registry", "ci/platforms.json", "platform registry path")
	capability := flag.String("capability", "", "required capability")
	githubOutput := flag.String("github-output", "", "GitHub Actions output file")
	flag.Parse()

	result, err := resolve(*registryPath, *capability)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve platform matrix: %v\n", err)
		os.Exit(1)
	}
	body, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encode platform matrix: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(body))
	if *githubOutput != "" {
		file, err := os.OpenFile(*githubOutput, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open GitHub output: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		if _, err := fmt.Fprintf(file, "matrix=%s\n", body); err != nil {
			fmt.Fprintf(os.Stderr, "write GitHub output: %v\n", err)
			os.Exit(1)
		}
	}
}

func resolve(path, capability string) (matrix, error) {
	if capability == "" {
		return matrix{}, errors.New("capability is required")
	}
	if _, ok := allowedCapabilities[capability]; !ok {
		return matrix{}, fmt.Errorf("unknown capability %q", capability)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return matrix{}, err
	}
	var cfg registry
	if err := json.Unmarshal(body, &cfg); err != nil {
		return matrix{}, err
	}
	if len(cfg.Platforms) == 0 {
		return matrix{}, errors.New("platform registry is empty")
	}

	seen := map[string]struct{}{}
	var include []matrixEntry
	for index, item := range cfg.Platforms {
		if err := validatePlatform(item); err != nil {
			return matrix{}, fmt.Errorf("platform %d: %w", index+1, err)
		}
		key := item.GOOS + "/" + item.GOARCH
		if _, ok := seen[key]; ok {
			return matrix{}, fmt.Errorf("duplicate platform %s", key)
		}
		seen[key] = struct{}{}
		if hasCapability(item, capability) {
			include = append(include, matrixEntry{
				GOOS:         item.GOOS,
				GOARCH:       item.GOARCH,
				Runner:       item.Runner,
				Capabilities: append([]string(nil), item.Capabilities...),
				Artifact:     item.GOOS + "-" + item.GOARCH,
			})
		}
	}
	if len(include) == 0 {
		return matrix{}, fmt.Errorf("no platform provides capability %q", capability)
	}
	return matrix{Include: include}, nil
}

func validatePlatform(item platform) error {
	if item.GOOS == "" || strings.ContainsAny(item.GOOS, "/\\") {
		return errors.New("goos must be a simple non-empty name")
	}
	if item.GOARCH == "" || strings.ContainsAny(item.GOARCH, "/\\") {
		return errors.New("goarch must be a simple non-empty name")
	}
	if strings.TrimSpace(item.Runner) == "" {
		return errors.New("runner is required")
	}
	if len(item.Capabilities) == 0 {
		return errors.New("at least one capability is required")
	}
	seen := map[string]struct{}{}
	for _, capability := range item.Capabilities {
		if _, ok := allowedCapabilities[capability]; !ok {
			return fmt.Errorf("unknown capability %q", capability)
		}
		if _, ok := seen[capability]; ok {
			return fmt.Errorf("duplicate capability %q", capability)
		}
		seen[capability] = struct{}{}
	}
	if hasCapability(item, "container") && item.GOOS != "linux" {
		return errors.New("container capability currently requires linux")
	}
	return nil
}

func hasCapability(item platform, want string) bool {
	values := sortedCopy(item.Capabilities)
	index := sort.SearchStrings(values, want)
	return index < len(values) && values[index] == want
}

func sortedCopy(values []string) []string {
	copyOf := append([]string(nil), values...)
	sort.Strings(copyOf)
	return copyOf
}
