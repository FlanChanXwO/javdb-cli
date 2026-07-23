package update

import (
	"fmt"
	"net/http"
	"net/url"
)

// NewReleaseHTTPClient creates the GitHub client used only by javdb update.
// A blank proxy preserves the Go transport's documented HTTPS_PROXY/ALL_PROXY behavior.
func NewReleaseHTTPClient(proxy string) (*http.Client, error) {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("default HTTP transport has unexpected type %T", http.DefaultTransport)
	}
	cloned := transport.Clone()
	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("parse update proxy: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return nil, fmt.Errorf("update proxy must use http or https, got %q", parsed.Scheme)
		}
		if parsed.Host == "" {
			return nil, fmt.Errorf("update proxy has no host")
		}
		cloned.Proxy = http.ProxyURL(parsed)
	}
	return &http.Client{Transport: cloned}, nil
}
