package client

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	storageroute "github.com/FlanChanXwO/javdb-cli/internal/storage/route"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// isolateHome points the platform home lookups at a fresh temp directory so config
// and device state never leak across tests or into a real user profile.
func isolateHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(dir))
	t.Setenv("HOMEPATH", strings.TrimPrefix(dir, filepath.VolumeName(dir)))
	return dir
}

// fakeAutoHost 构造注入 fake cache/selector 的 autoHost，并记录调用。
func fakeAutoHost(t *testing.T, host string, reused bool) (autoHost, func() string, func() *javdb.AutoHostOptions) {
	t.Helper()
	var saved string
	var captured javdb.AutoHostOptions
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			return storageroute.Document{}, false, nil
		},
		saveCache: func(path string, doc storageroute.Document) error {
			saved = doc.Host
			return nil
		},
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			captured = opts
			return javdb.AutoHostResult{Host: host, ReusedPreferred: reused}, nil
		},
	}
	return ah, func() string { return saved }, func() *javdb.AutoHostOptions { return &captured }
}

func TestNewBuildsFixedHostClientWithoutNetwork(t *testing.T) {
	isolateHome(t)
	c, err := New(&invocation.RootOptions{Host: settings.HostMirror}, "")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if c == nil {
		t.Fatal("New returned nil client")
	}
	if got := c.Token(); got != "" {
		t.Fatalf("Token() = %q, want empty", got)
	}
}

func TestNewRejectsInvalidHost(t *testing.T) {
	isolateHome(t)
	if _, err := New(&invocation.RootOptions{Host: "bogus"}, ""); err == nil {
		t.Fatal("New accepted an invalid host")
	}
}

func TestNewCreatesAndPersistsDeviceUUID(t *testing.T) {
	home := isolateHome(t)

	first, err := New(&invocation.RootOptions{Host: settings.HostMirror}, "")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	id1 := first.API().DeviceUUID()
	if id1 == "" {
		t.Fatal("DeviceUUID() is empty")
	}
	devicePath := filepath.Join(home, ".javdb-cli", "device_uuid")
	if data, err := os.ReadFile(devicePath); err != nil || strings.TrimSpace(string(data)) != id1 {
		t.Fatalf("device_uuid file = %q, %v; want %q", data, err, id1)
	}

	second, err := New(&invocation.RootOptions{Host: settings.HostMirror}, "")
	if err != nil {
		t.Fatalf("second New error = %v", err)
	}
	if got := second.API().DeviceUUID(); got != id1 {
		t.Fatalf("reused DeviceUUID = %q, want %q", got, id1)
	}
}

func TestNewCarriesToken(t *testing.T) {
	isolateHome(t)
	c, err := New(&invocation.RootOptions{Host: settings.HostMirror}, "tok")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if got := c.Token(); got != "tok" {
		t.Fatalf("Token() = %q, want tok", got)
	}
}

func TestBuildClientCarriesTokenAndDevice(t *testing.T) {
	rt := settings.Runtime{Host: settings.HostMirror, BaseURL: "https://jdforrepam.com", DeviceUUID: "dev-1", Lang: "en"}
	c, err := buildClient(rt, "https://selected.example", "tok")
	if err != nil {
		t.Fatalf("buildClient error = %v", err)
	}
	if got := c.Token(); got != "tok" {
		t.Fatalf("Token() = %q, want tok", got)
	}
	if got := c.API().DeviceUUID(); got != "dev-1" {
		t.Fatalf("DeviceUUID() = %q, want dev-1", got)
	}
}

func TestResolveBaseURLFixedBypassesCacheAndSelector(t *testing.T) {
	called := false
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			called = true
			return storageroute.Document{}, false, nil
		},
		saveCache: func(path string, doc storageroute.Document) error {
			called = true
			return nil
		},
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			called = true
			return javdb.AutoHostResult{}, nil
		},
	}
	rt := settings.Runtime{Host: settings.HostMirror, BaseURL: "https://jdforrepam.com"}
	baseURL, err := resolveBaseURL(rt, ah)
	if err != nil {
		t.Fatalf("resolveBaseURL error = %v", err)
	}
	if baseURL != rt.BaseURL {
		t.Fatalf("baseURL = %q, want %q", baseURL, rt.BaseURL)
	}
	if called {
		t.Fatal("fixed host must bypass route cache and selector")
	}
}

func TestResolveBaseURLAutoCacheHitReusesWithoutSaving(t *testing.T) {
	const cached = "https://cached.example"
	saveCalled := false
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			return storageroute.Document{Host: cached}, true, nil
		},
		saveCache: func(path string, doc storageroute.Document) error {
			saveCalled = true
			return nil
		},
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			if opts.PreferredHost != cached {
				t.Fatalf("PreferredHost = %q, want %q", opts.PreferredHost, cached)
			}
			return javdb.AutoHostResult{Host: cached, ReusedPreferred: true}, nil
		},
	}
	baseURL, err := resolveBaseURL(settings.Runtime{Host: settings.HostAuto}, ah)
	if err != nil {
		t.Fatalf("resolveBaseURL error = %v", err)
	}
	if baseURL != cached {
		t.Fatalf("baseURL = %q, want %q", baseURL, cached)
	}
	if saveCalled {
		t.Fatal("cache reuse must not rewrite route cache")
	}
}

func TestResolveBaseURLAutoCacheMissSelectsAndSaves(t *testing.T) {
	const selected = "https://selected.example"
	ah, savedHost, _ := fakeAutoHost(t, selected, false)
	baseURL, err := resolveBaseURL(settings.Runtime{Host: settings.HostAuto}, ah)
	if err != nil {
		t.Fatalf("resolveBaseURL error = %v", err)
	}
	if baseURL != selected {
		t.Fatalf("baseURL = %q, want %q", baseURL, selected)
	}
	if got := savedHost(); got != selected {
		t.Fatalf("saved host = %q, want %q", got, selected)
	}
}

func TestResolveBaseURLPassesProxyDeviceLang(t *testing.T) {
	rt := settings.Runtime{
		Host:       settings.HostAuto,
		Proxy:      "http://proxy.example:7890",
		DeviceUUID: "dev-123",
		Lang:       "zh",
	}
	ah, _, captured := fakeAutoHost(t, "https://selected.example", false)
	if _, err := resolveBaseURL(rt, ah); err != nil {
		t.Fatalf("resolveBaseURL error = %v", err)
	}
	opts := captured()
	if opts.Proxy != rt.Proxy || opts.DeviceUUID != rt.DeviceUUID || opts.Lang != rt.Lang {
		t.Fatalf("AutoHostOptions = %+v, want proxy/device/lang passthrough", opts)
	}
	if opts.PreferredHost != "" {
		t.Fatalf("PreferredHost = %q, want empty on cache miss", opts.PreferredHost)
	}
}

func TestResolveBaseURLAutoCacheFailureErrors(t *testing.T) {
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			return storageroute.Document{}, false, errors.New("corrupt cache")
		},
		saveCache: func(path string, doc storageroute.Document) error { return nil },
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			return javdb.AutoHostResult{}, nil
		},
	}
	_, err := resolveBaseURL(settings.Runtime{Host: settings.HostAuto}, ah)
	if err == nil || !strings.Contains(err.Error(), "load route cache") {
		t.Fatalf("error = %v, want load route cache failure", err)
	}
}

func TestResolveBaseURLAutoSelectorFailureErrors(t *testing.T) {
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			return storageroute.Document{}, false, nil
		},
		saveCache: func(path string, doc storageroute.Document) error { return nil },
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			return javdb.AutoHostResult{}, errors.New("all hosts offline")
		},
	}
	_, err := resolveBaseURL(settings.Runtime{Host: settings.HostAuto}, ah)
	if err == nil || !strings.Contains(err.Error(), "auto select host") {
		t.Fatalf("error = %v, want auto select host failure", err)
	}
}

func TestResolveBaseURLAutoSaveFailureErrors(t *testing.T) {
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			return storageroute.Document{}, false, nil
		},
		saveCache: func(path string, doc storageroute.Document) error {
			return errors.New("disk full")
		},
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			return javdb.AutoHostResult{Host: "https://selected.example"}, nil
		},
	}
	_, err := resolveBaseURL(settings.Runtime{Host: settings.HostAuto}, ah)
	if err == nil || !strings.Contains(err.Error(), "save route cache") {
		t.Fatalf("error = %v, want save route cache failure", err)
	}
}

// TestResolveRuntimeReturnsDeviceUUIDError 验证 device_uuid 路径/读写错误显式返回，
// 不会被吞掉导致 auto probe 与业务 client 使用不同随机 UUID。
func TestResolveRuntimeReturnsDeviceUUIDError(t *testing.T) {
	home := isolateHome(t)
	dir := filepath.Join(home, ".javdb-cli")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	// 用目录占位 device_uuid，使 LoadOrCreateDeviceUUID 的写入失败。
	if err := os.MkdirAll(filepath.Join(dir, "device_uuid"), 0o700); err != nil {
		t.Fatalf("MkdirAll(device_uuid) error = %v", err)
	}
	if _, err := resolveRuntime(&invocation.RootOptions{Host: settings.HostMirror}); err == nil {
		t.Fatal("resolveRuntime swallowed device uuid creation error")
	}
}

// TestResolveRuntimeRejectsProxyBeforeDeviceUUID 验证非法 proxy 在 device UUID provision
// 之前完成无副作用校验：不留下 device_uuid 本机状态副作用。
func TestResolveRuntimeRejectsProxyBeforeDeviceUUID(t *testing.T) {
	home := isolateHome(t)
	if _, err := resolveRuntime(&invocation.RootOptions{Host: settings.HostMirror, Proxy: "://bad"}); err == nil || !strings.Contains(err.Error(), "proxy") {
		t.Fatalf("resolveRuntime error = %v, want proxy validation error", err)
	}
	devicePath := filepath.Join(home, ".javdb-cli", "device_uuid")
	if _, statErr := os.Stat(devicePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("device_uuid created despite invalid proxy: %v", statErr)
	}
}

// TestResolveClientCreatesConfigBeforeSelection 验证：参数完全有效的首次 auto 命令即使网络
// 选线失败，也会先创建基线配置（"正常可执行命令触发创建"契约）。
func TestResolveClientCreatesConfigBeforeSelection(t *testing.T) {
	home := isolateHome(t)
	ah := autoHost{
		loadCache: func(path string) (storageroute.Document, bool, error) {
			return storageroute.Document{}, false, nil
		},
		saveCache: func(path string, doc storageroute.Document) error { return nil },
		selectHost: func(ctx context.Context, opts javdb.AutoHostOptions) (javdb.AutoHostResult, error) {
			return javdb.AutoHostResult{}, errors.New("selection failed")
		},
	}
	// 默认 auto host；selectHost 注入失败以模拟断网选线失败。
	_, _, err := resolveClientWithAutoHost(&invocation.RootOptions{}, ah)
	if err == nil || !strings.Contains(err.Error(), "auto select host") {
		t.Fatalf("error = %v, want selection failure", err)
	}
	configPath := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(configPath); statErr != nil {
		t.Fatalf("config.toml not created before failing selection: %v", statErr)
	}
}
