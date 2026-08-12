package settings

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

// reverse_search 区域默认值；超时/重试沿用参考服务行为且可配置。
const (
	DefaultReverseSearchSource         = "builtin"
	DefaultReverseSearchCacheTTL       = "720h"
	DefaultReverseSearchRetries        = 3
	DefaultReverseSearchRetryWait      = "30s"
	DefaultReverseSearchRequestTimeout = "60s"
)

// ReverseSearchSource 是 config.toml 中 [[reverse_search.sources]] 的一项。
// Headers 的值只允许静态文本与 ${ENV:NAME} 引用，禁止明文 secret。
type ReverseSearchSource struct {
	Name    string            `toml:"name"`
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
}

// ReverseSearchSettings 是 [reverse_search] 表的 typed 视图。
type ReverseSearchSettings struct {
	DefaultSource  string                `toml:"default_source"`
	Cache          *bool                 `toml:"cache"`
	CacheTTL       string                `toml:"cache_ttl"`
	Retries        int                   `toml:"retries"`
	RetryWait      string                `toml:"retry_wait"`
	RequestTimeout string                `toml:"request_timeout"`
	Sources        []ReverseSearchSource `toml:"sources"`
}

// applyDefaults 把未显式配置的标量填为默认值。
func (r *ReverseSearchSettings) applyDefaults() {
	if r.DefaultSource == "" {
		r.DefaultSource = DefaultReverseSearchSource
	}
	if r.Cache == nil {
		r.Cache = boolPointer(true)
	}
	if r.CacheTTL == "" {
		r.CacheTTL = DefaultReverseSearchCacheTTL
	}
	if r.Retries == 0 {
		r.Retries = DefaultReverseSearchRetries
	}
	if r.RetryWait == "" {
		r.RetryWait = DefaultReverseSearchRetryWait
	}
	if r.RequestTimeout == "" {
		r.RequestTimeout = DefaultReverseSearchRequestTimeout
	}
}

// CacheEnabled 报告反搜文件缓存是否启用（默认 true）。
func (r ReverseSearchSettings) CacheEnabled() bool {
	return r.Cache == nil || *r.Cache
}

func boolPointer(value bool) *bool {
	return &value
}

// ResolvedReverseSearch 是校验并展开后的反搜配置；Headers 已完成环境展开。
type ResolvedReverseSearch struct {
	DefaultSource  string
	Cache          bool
	CacheTTL       time.Duration
	Retries        int
	RetryWait      time.Duration
	RequestTimeout time.Duration
	Sources        []ReverseSearchSource
}

// envReferencePattern 匹配 ${ENV:NAME} 引用。
var envReferencePattern = regexp.MustCompile(`\$\{ENV:([A-Za-z_][A-Za-z0-9_]*)\}`)

// ExpandEnvValue 展开 header 值中的 ${ENV:NAME} 引用。缺失或空的环境变量
// 报错只包含变量名，绝不包含展开后的值。
func ExpandEnvValue(value string, getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = envLookup
	}
	matches := envReferencePattern.FindAllStringSubmatch(value, -1)
	expanded := value
	for _, match := range matches {
		name := match[1]
		replacement := getenv(name)
		if replacement == "" {
			return "", fmt.Errorf("environment variable %s referenced by a reverse search header is not set", name)
		}
		expanded = strings.ReplaceAll(expanded, "${ENV:"+name+"}", replacement)
	}
	return expanded, nil
}

func envLookup(name string) string {
	return os.Getenv(name)
}

// sourceNamePattern 约束 source 名只含字母、数字、- 与 _，避免缓存文件名
// 消毒后不同 source 互相覆盖。
var sourceNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ResolveReverseSearch 校验 reverse_search 区域并展开 header 环境引用。
func ResolveReverseSearch(s Settings, getenv func(string) string) (ResolvedReverseSearch, error) {
	rs := s.ReverseSearch
	rs.applyDefaults()
	resolved := ResolvedReverseSearch{
		DefaultSource: rs.DefaultSource,
		Cache:         rs.CacheEnabled(),
		Sources:       make([]ReverseSearchSource, 0, len(rs.Sources)),
	}
	if resolved.DefaultSource == "" {
		resolved.DefaultSource = DefaultReverseSearchSource
	}
	ttl, err := time.ParseDuration(rs.CacheTTL)
	if err != nil || ttl <= 0 {
		return ResolvedReverseSearch{}, fmt.Errorf("reverse_search.cache_ttl %q must be a positive duration", rs.CacheTTL)
	}
	resolved.CacheTTL = ttl
	retries := rs.Retries
	if retries == 0 {
		retries = DefaultReverseSearchRetries
	}
	if retries < 1 {
		return ResolvedReverseSearch{}, fmt.Errorf("reverse_search.retries must be at least 1, got %d", retries)
	}
	resolved.Retries = retries
	wait, err := time.ParseDuration(rs.RetryWait)
	if err != nil || wait <= 0 {
		return ResolvedReverseSearch{}, fmt.Errorf("reverse_search.retry_wait %q must be a positive duration", rs.RetryWait)
	}
	resolved.RetryWait = wait
	timeout, err := time.ParseDuration(rs.RequestTimeout)
	if err != nil || timeout <= 0 {
		return ResolvedReverseSearch{}, fmt.Errorf("reverse_search.request_timeout %q must be a positive duration", rs.RequestTimeout)
	}
	resolved.RequestTimeout = timeout

	seen := map[string]bool{}
	for index, source := range rs.Sources {
		if strings.TrimSpace(source.Name) == "" {
			return ResolvedReverseSearch{}, fmt.Errorf("reverse_search.sources[%d] must have a name", index)
		}
		if !sourceNamePattern.MatchString(source.Name) {
			return ResolvedReverseSearch{}, fmt.Errorf("reverse_search source name %q may only contain letters, digits, - and _", source.Name)
		}
		if source.Name == "builtin" {
			return ResolvedReverseSearch{}, fmt.Errorf("reverse_search source name %q is reserved for the builtin provider", source.Name)
		}
		if seen[source.Name] {
			return ResolvedReverseSearch{}, fmt.Errorf("reverse_search source name %q is duplicated", source.Name)
		}
		seen[source.Name] = true
		parsed, err := url.Parse(source.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return ResolvedReverseSearch{}, fmt.Errorf("reverse_search source %q URL must be absolute HTTP(S), got %q", source.Name, source.URL)
		}
		expanded := ReverseSearchSource{Name: source.Name, URL: source.URL}
		if len(source.Headers) > 0 {
			expanded.Headers = make(map[string]string, len(source.Headers))
			for name, value := range source.Headers {
				expandedValue, err := ExpandEnvValue(value, getenv)
				if err != nil {
					return ResolvedReverseSearch{}, fmt.Errorf("reverse_search source %q header %q: %w", source.Name, name, err)
				}
				expanded.Headers[name] = expandedValue
			}
		}
		resolved.Sources = append(resolved.Sources, expanded)
	}
	if resolved.DefaultSource != "builtin" && !seen[resolved.DefaultSource] {
		return ResolvedReverseSearch{}, fmt.Errorf("reverse_search.default_source %q must be builtin or a defined source", resolved.DefaultSource)
	}
	return resolved, nil
}
