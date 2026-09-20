package settings

import "fmt"

const DefaultAssetProbeConcurrency = 4

// AssetProbeSettings 是 config.toml 中可选的 [assets.probe] 配置。
type AssetProbeSettings struct {
	Enabled     *bool `toml:"enabled"`
	Concurrency *int  `toml:"concurrency"`
}

// AssetsSettings 是 assets 配置表。
type AssetsSettings struct {
	Probe AssetProbeSettings `toml:"probe"`
}

// ResolvedAssetProbe 是应用默认值并通过校验后的 probe 配置。
type ResolvedAssetProbe struct {
	Enabled     bool
	Concurrency int
}

// ResolveAssetProbe 返回当前有效的资产探测配置。
func ResolveAssetProbe(s Settings) (ResolvedAssetProbe, error) {
	resolved := ResolvedAssetProbe{
		Enabled:     true,
		Concurrency: DefaultAssetProbeConcurrency,
	}
	if s.Assets.Probe.Enabled != nil {
		resolved.Enabled = *s.Assets.Probe.Enabled
	}
	if s.Assets.Probe.Concurrency != nil {
		resolved.Concurrency = *s.Assets.Probe.Concurrency
	}
	if resolved.Concurrency <= 0 {
		return ResolvedAssetProbe{}, fmt.Errorf("assets.probe.concurrency must be positive, got %d", resolved.Concurrency)
	}
	return resolved, nil
}
