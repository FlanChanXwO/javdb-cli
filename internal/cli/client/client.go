// Package client 统一配置解析、SDK client 创建与 required/optional 认证生命周期。
package client

import (
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New 读取配置、校验命令行 host、解析运行时，并构造携带给定 token 的公开 SDK client。
// 在缺失 device UUID 时沿用 LoadOrCreateDeviceUUID 的路径与错误处理。
func New(options *invocation.RootOptions, token string) (*javdb.Client, error) {
	rt, err := resolveRuntime(options)
	if err != nil {
		return nil, err
	}
	return buildClient(rt, token)
}

// resolveRuntime 解析配置文件、环境变量和根 flags，并补齐 device UUID。
func resolveRuntime(options *invocation.RootOptions) (settings.Runtime, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return settings.Runtime{}, err
	}
	file, err := settings.LoadFile(path)
	if err != nil {
		return settings.Runtime{}, err
	}
	if err := settings.ValidateHost(options.Host); err != nil {
		return settings.Runtime{}, err
	}
	runtimeConfig := settings.Resolve(file, options.Host, options.Proxy, nil)
	if runtimeConfig.DeviceUUID == "" {
		devicePath, err := paths.DeviceUUIDPath()
		if err == nil {
			if id, err := javdb.LoadOrCreateDeviceUUID(devicePath); err == nil {
				runtimeConfig.DeviceUUID = id
			}
		}
	}
	return runtimeConfig, nil
}

// buildClient 根据已解析的 runtime 构造公开 SDK client。
func buildClient(rt settings.Runtime, token string) (*javdb.Client, error) {
	return javdb.New(
		javdb.WithHost(rt.BaseURL),
		javdb.WithProxy(rt.Proxy),
		javdb.WithToken(token),
		javdb.WithDeviceUUID(rt.DeviceUUID),
		javdb.WithLang(rt.Lang),
	)
}
