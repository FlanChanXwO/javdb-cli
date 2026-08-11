// Package client 统一配置解析、SDK client 创建与 required/optional 认证生命周期。
package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	storageroute "github.com/FlanChanXwO/javdb-cli/internal/storage/route"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// autoHost 是自动选线的可注入依赖集合，便于测试注入 fake cache/selector；不引入可变的
// 全局替换，生产依赖只读。
type autoHost struct {
	loadCache  func(path string) (storageroute.Document, bool, error)
	saveCache  func(path string, doc storageroute.Document) error
	selectHost func(ctx context.Context, options javdb.AutoHostOptions) (javdb.AutoHostResult, error)
}

// productionAutoHost 是生产依赖：真实 route cache 与公开 SDK selector。
var productionAutoHost = autoHost{
	loadCache:  storageroute.Load,
	saveCache:  storageroute.Save,
	selectHost: javdb.SelectAutoHost,
}

// New 读取配置、校验命令行 host 与 proxy、解析运行时，并在 auto 时先完成线路选择，再构造
// 携带给定 token 的公开 SDK client。固定 host 完全绕过 route cache 与 selector。基线配置在
// host/proxy 无副作用校验与 device UUID provision 之后创建；无效 host/proxy 不落盘，选线
// 失败也不回滚已创建的本机基线状态。
func New(options *invocation.RootOptions, token string) (*javdb.Client, error) {
	rt, baseURL, err := resolveClient(options)
	if err != nil {
		return nil, err
	}
	return buildClient(rt, baseURL, token)
}

// resolveClient 解析运行时（校验 host/proxy、补齐 device UUID）、创建基线配置，并确定
// baseURL：auto 走 route cache + 公开 SDK selector，固定 host 直接使用 runtime 的 BaseURL。
func resolveClient(options *invocation.RootOptions) (settings.Runtime, string, error) {
	return resolveClientWithAutoHost(options, productionAutoHost)
}

// resolveClientWithAutoHost 是 resolveClient 的可注入核心，供测试注入 fake cache/selector。
func resolveClientWithAutoHost(options *invocation.RootOptions, ah autoHost) (settings.Runtime, string, error) {
	rt, err := resolveRuntime(options)
	if err != nil {
		return settings.Runtime{}, "", err
	}
	// 无副作用的 host/proxy 校验通过后、可能失败的网络选线前创建基线配置。
	if err := paths.EnsureDefaultConfigFile(); err != nil {
		return settings.Runtime{}, "", err
	}
	baseURL, err := resolveBaseURL(rt, ah)
	if err != nil {
		return settings.Runtime{}, "", err
	}
	return rt, baseURL, nil
}

// resolveRuntime 解析配置文件、环境变量和根 flags，校验 host 与 proxy，并补齐 device UUID。
func resolveRuntime(options *invocation.RootOptions) (settings.Runtime, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return settings.Runtime{}, err
	}
	file, err := settings.LoadFile(path)
	if err != nil {
		return settings.Runtime{}, err
	}
	runtimeConfig, err := settings.Resolve(file, options.Host, options.Proxy, nil)
	if err != nil {
		return settings.Runtime{}, err
	}
	// proxy 在 device UUID provision 之前做无副作用校验，参数校验失败不产生本机状态副作用。
	if err := validateProxy(runtimeConfig.Proxy); err != nil {
		return settings.Runtime{}, err
	}
	if runtimeConfig.DeviceUUID == "" {
		// device_uuid 读取/创建失败必须显式返回，否则 auto probe 与业务 client 会各自生成
		// 不同随机 UUID，破坏"同一 effective device UUID"契约。
		devicePath, err := paths.DeviceUUIDPath()
		if err != nil {
			return settings.Runtime{}, err
		}
		id, err := javdb.LoadOrCreateDeviceUUID(devicePath)
		if err != nil {
			return settings.Runtime{}, err
		}
		runtimeConfig.DeviceUUID = id
	}
	return runtimeConfig, nil
}

// validateProxy 无副作用地校验 proxy URL（不构造 transport、不写本机状态），与 transport
// 的校验一致：必须是带 host 的绝对 http/https/socks5 代理 URL。
func validateProxy(proxy string) error {
	if strings.TrimSpace(proxy) == "" {
		return nil
	}
	u, err := url.Parse(proxy)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("invalid proxy URL %q", proxy)
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5", "socks5h":
		return nil
	default:
		return fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}
}

// resolveBaseURL 对固定 host（mirror/main/URL）直接返回 runtime 的 BaseURL；对 auto 读取
// route cache，用公开 SDK selector 验证缓存或重测线路，必要时在构造业务 client 前持久化
// 选中 URL。cache/selector/save 错误原样返回，不伪装为 miss。
func resolveBaseURL(rt settings.Runtime, ah autoHost) (string, error) {
	if rt.Host != settings.HostAuto {
		return rt.BaseURL, nil
	}
	path, err := paths.RouteCachePath()
	if err != nil {
		return "", err
	}
	preferred := ""
	if cached, ok, err := ah.loadCache(path); err != nil {
		return "", fmt.Errorf("load route cache: %w", err)
	} else if ok {
		preferred = cached.Host
	}
	result, err := ah.selectHost(context.Background(), javdb.AutoHostOptions{
		PreferredHost: preferred,
		Proxy:         rt.Proxy,
		DeviceUUID:    rt.DeviceUUID,
		Lang:          rt.Lang,
	})
	if err != nil {
		return "", fmt.Errorf("auto select host: %w", err)
	}
	if !result.ReusedPreferred {
		if err := ah.saveCache(path, storageroute.Document{Host: result.Host}); err != nil {
			return "", fmt.Errorf("save route cache: %w", err)
		}
	}
	return result.Host, nil
}

// buildClient 根据已解析的 runtime 与选定 baseURL 构造公开 SDK client。
func buildClient(rt settings.Runtime, baseURL, token string) (*javdb.Client, error) {
	return javdb.New(
		javdb.WithHost(baseURL),
		javdb.WithProxy(rt.Proxy),
		javdb.WithToken(token),
		javdb.WithDeviceUUID(rt.DeviceUUID),
		javdb.WithLang(rt.Lang),
	)
}
