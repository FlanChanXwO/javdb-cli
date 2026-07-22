// Package httpx wraps bogdanfinn/tls-client for Chrome-like TLS fingerprints.
package httpx

import (
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// Client is a thin wrapper around tls-client with proxy support.
type Client struct {
	inner   tls_client.HttpClient
	proxy   string
	timeout time.Duration
}

// Options configures a new Client.
type Options struct {
	Proxy   string
	Timeout time.Duration
}

// New builds a Chrome-profile tls-client.
func New(opts Options) (*Client, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 20 * time.Second
	}
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(int(opts.Timeout.Seconds())),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
	}
	if opts.Proxy != "" {
		options = append(options, tls_client.WithProxyUrl(opts.Proxy))
	}
	inner, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		return nil, fmt.Errorf("tls-client: %w", err)
	}
	return &Client{inner: inner, proxy: opts.Proxy, timeout: opts.Timeout}, nil
}

// Do executes a request.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.inner.Do(req)
}

// Get is a convenience GET.
func (c *Client) Get(urlStr string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.Do(req)
}

// PostForm posts application/x-www-form-urlencoded body.
func (c *Client) PostForm(urlStr string, form url.Values, headers map[string]string) (*http.Response, error) {
	body := strings.NewReader(form.Encode())
	req, err := http.NewRequest(http.MethodPost, urlStr, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.Do(req)
}

// Delete issues a DELETE with query params already in urlStr.
func (c *Client) Delete(urlStr string, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, urlStr, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.Do(req)
}

// ReadAll reads and closes the response body.
func ReadAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
