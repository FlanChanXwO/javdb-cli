package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const defaultReleasesEndpoint = "https://api.github.com/repos/" + githubRepository + "/releases?per_page=100"

// ReleaseChecker resolves the highest compatible published GitHub Release.
type ReleaseChecker interface {
	Check(context.Context, bool) (*Release, error)
}

// ReleaseClientOptions exposes the API endpoint for deterministic tests.
type ReleaseClientOptions struct {
	HTTPClient *http.Client
	Endpoint   string
}

// GitHubReleaseClient reads GitHub's public Releases API.
type GitHubReleaseClient struct {
	httpClient *http.Client
	endpoint   string
}

// NewGitHubReleaseClient constructs an unauthenticated Release reader.
func NewGitHubReleaseClient(options ReleaseClientOptions) (*GitHubReleaseClient, error) {
	if options.HTTPClient == nil {
		return nil, fmt.Errorf("release HTTP client is required")
	}
	endpoint := options.Endpoint
	if endpoint == "" {
		endpoint = defaultReleasesEndpoint
	}
	return &GitHubReleaseClient{httpClient: options.HTTPClient, endpoint: endpoint}, nil
}

// Check returns the highest SemVer release. GitHub page links are followed until
// exhausted so a repository with more than one API page is not silently truncated.
func (c *GitHubReleaseClient) Check(ctx context.Context, includePrerelease bool) (*Release, error) {
	if c == nil || c.httpClient == nil {
		return nil, fmt.Errorf("release client is not configured")
	}
	endpoint := c.endpoint
	var selected *Release
	var selectedVersion semanticVersion
	for endpoint != "" {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("create GitHub Releases request: %w", err)
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("User-Agent", "javdb-cli")
		response, err := c.httpClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("request GitHub Releases: %w", err)
		}
		if response.StatusCode != http.StatusOK {
			_ = response.Body.Close()
			return nil, fmt.Errorf("GitHub Releases returned HTTP %s", response.Status)
		}
		var page []struct {
			Release
			Draft bool `json:"draft"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&page)
		next := nextReleasePage(response.Header.Get("Link"))
		closeErr := response.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode GitHub Releases response: %w", decodeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close GitHub Releases response: %w", closeErr)
		}
		for _, candidate := range page {
			if candidate.Draft {
				continue
			}
			parsed, err := parseSemanticVersion(candidate.TagName)
			if err != nil {
				continue
			}
			// GitHub normally keeps this flag aligned with the tag. Check both so a
			// malformed Release record cannot make --check advertise a prerelease.
			if !includePrerelease && (candidate.Prerelease || parsed.isPrerelease()) {
				continue
			}
			if selected == nil || parsed.compare(selectedVersion) > 0 {
				copy := candidate.Release
				selected = &copy
				selectedVersion = parsed
			}
		}
		endpoint = next
	}
	return selected, nil
}

func nextReleasePage(linkHeader string) string {
	for _, link := range strings.Split(linkHeader, ",") {
		parts := strings.Split(link, ";")
		if len(parts) < 2 || !strings.Contains(parts[1], `rel="next"`) {
			continue
		}
		value := strings.TrimSpace(parts[0])
		if strings.HasPrefix(value, "<") && strings.HasSuffix(value, ">") {
			return strings.TrimSuffix(strings.TrimPrefix(value, "<"), ">")
		}
	}
	return ""
}
