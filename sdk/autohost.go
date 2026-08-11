package javdb

import (
	"context"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"
)

// AutoHostOptions 配置显式自动线路选择。PreferredHost 是调用方提供的待验证候选
// （通常来自上次选线的 route cache）；SDK 不负责持久化。
type AutoHostOptions struct {
	PreferredHost string
	Proxy         string
	DeviceUUID    string
	Timeout       time.Duration
	Lang          string
}

// AutoHostResult 是自动选线结果。Latency 是缓存验证成功或全量测速赢家的单次 /startup
// 耗时。调用方用 javdb.New(WithHost(result.Host), ...) 构造业务客户端。
type AutoHostResult struct {
	Host            string
	Latency         time.Duration
	ReusedPreferred bool
}

// SelectAutoHost 通过真实 /startup 探测选择最快的 JavDB App API host。
//
// 它显式联网选线，返回具体 URL；不会让 javdb.New(WithHost("auto")) 自动联网。调用方应先
// 验证 route cache 中的 PreferredHost，成功后（ReusedPreferred==true）无需写入新结果；
// 重测选择时再把 result.Host 持久化。
func SelectAutoHost(ctx context.Context, options AutoHostOptions) (AutoHostResult, error) {
	opts := appapi.AutoHostOptions(options)
	probe, err := appapi.NewAutoHostProbe(opts)
	if err != nil {
		return AutoHostResult{}, err
	}
	result, err := appapi.SelectAutoHost(ctx, opts, probe)
	if err != nil {
		return AutoHostResult{}, err
	}
	return AutoHostResult{
		Host:            result.Host,
		Latency:         result.Latency,
		ReusedPreferred: result.ReusedPreferred,
	}, nil
}
