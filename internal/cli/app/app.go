// Package app 集中管理 CLI 的 IO、运行时配置、SDK client 与本机认证依赖。
package app

import (
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// Flags 是根命令的 persistent flags。
type Flags struct {
	Proxy string
	Host  string
}

// IO 是命令共享的标准输入、标准输出和错误输出。
type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// NewIO 构造命令共享 IO。
func NewIO(in io.Reader, out, err io.Writer) *IO {
	return &IO{In: in, Out: out, Err: err}
}

// LoadRuntime 解析配置文件、环境变量和根 flags，并补齐 device UUID。
func LoadRuntime(flags *Flags) (settings.Runtime, error) {
	path, err := paths.ConfigPath()
	if err != nil {
		return settings.Runtime{}, err
	}
	file, err := settings.LoadFile(path)
	if err != nil {
		return settings.Runtime{}, err
	}
	if err := settings.ValidateHost(flags.Host); err != nil {
		return settings.Runtime{}, err
	}
	runtimeConfig := settings.Resolve(file, flags.Host, flags.Proxy, nil)
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

// OpenAuth 打开默认认证文件并确保其目录存在。
func OpenAuth() (*auth.FileStore, *auth.Store, error) {
	path, err := paths.AuthPath()
	if err != nil {
		return nil, nil, err
	}
	if _, err := paths.EnsureDir(); err != nil {
		return nil, nil, err
	}
	return auth.Open(path)
}

// NewClient 根据解析后的 runtime 创建公开 SDK client。
func NewClient(runtimeConfig settings.Runtime, token string) (*javdb.Client, error) {
	return javdb.New(
		javdb.WithHost(runtimeConfig.BaseURL),
		javdb.WithProxy(runtimeConfig.Proxy),
		javdb.WithToken(token),
		javdb.WithDeviceUUID(runtimeConfig.DeviceUUID),
		javdb.WithLang(runtimeConfig.Lang),
	)
}
