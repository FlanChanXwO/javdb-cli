package settings

import (
	"fmt"
	"time"
)

// assets.probe 区域默认值(计划 #3)。整个 [assets.probe] 表不存在时,
// 等效于以下默认配置;只配置部分字段时,其余字段继续使用默认值。
const (
	DefaultAssetsProbeEnabled              = true
	DefaultAssetsProbeConcurrency          = 4
	DefaultAssetsProbeImageMaxBytes        = 65536
	DefaultAssetsProbePlaylistMaxBytes     = 262144
	DefaultAssetsProbeVideoSegmentMaxBytes = 262144
	DefaultAssetsProbeTimeout              = "10s"
)

// AssetsProbeSettings 是 [assets.probe] 表的 typed 视图。
// Cache 指针模式沿用 reverse_search 区域的既有实现:指针为 nil 表示
// 未显式配置,此时默认值生效。
type AssetsProbeSettings struct {
	Enabled              *bool   `toml:"enabled"`
	Concurrency          *int    `toml:"concurrency"`
	ImageMaxBytes        *int64  `toml:"image_max_bytes"`
	PlaylistMaxBytes     *int64  `toml:"playlist_max_bytes"`
	VideoSegmentMaxBytes *int64  `toml:"video_segment_max_bytes"`
	Timeout              *string `toml:"timeout"`
}

// applyDefaults 把未显式配置的标量填为默认值。
// 指针语义区分"未配置"(填默认)与"显式配置"(原样保留,交给 validate)。
func (a *AssetsProbeSettings) applyDefaults() {
	if a.Enabled == nil {
		a.Enabled = boolPointer(DefaultAssetsProbeEnabled)
	}
	if a.Concurrency == nil {
		a.Concurrency = intPointer(DefaultAssetsProbeConcurrency)
	}
	if a.ImageMaxBytes == nil {
		a.ImageMaxBytes = int64Pointer(DefaultAssetsProbeImageMaxBytes)
	}
	if a.PlaylistMaxBytes == nil {
		a.PlaylistMaxBytes = int64Pointer(DefaultAssetsProbePlaylistMaxBytes)
	}
	if a.VideoSegmentMaxBytes == nil {
		a.VideoSegmentMaxBytes = int64Pointer(DefaultAssetsProbeVideoSegmentMaxBytes)
	}
	if a.Timeout == nil {
		a.Timeout = stringPointer(DefaultAssetsProbeTimeout)
	}
}

func intPointer(value int) *int          { return &value }
func int64Pointer(value int64) *int64    { return &value }
func stringPointer(value string) *string { return &value }

// ConcurrencyValue 返回生效的 probe 并发。
func (a AssetsProbeSettings) ConcurrencyValue() int { return *a.Concurrency }

// ImageMaxBytesValue 返回生效的图片 probe 预算。
func (a AssetsProbeSettings) ImageMaxBytesValue() int64 { return *a.ImageMaxBytes }

// PlaylistMaxBytesValue 返回生效的 playlist probe 预算。
func (a AssetsProbeSettings) PlaylistMaxBytesValue() int64 { return *a.PlaylistMaxBytes }

// VideoSegmentMaxBytesValue 返回生效的 segment probe 预算。
func (a AssetsProbeSettings) VideoSegmentMaxBytesValue() int64 { return *a.VideoSegmentMaxBytes }

// TimeoutValue 返回生效的 probe 超时。
func (a AssetsProbeSettings) TimeoutValue() string { return *a.Timeout }

// EnabledValue 报告 metadata probe 是否启用(默认 true)。
func (a AssetsProbeSettings) EnabledValue() bool {
	return a.Enabled == nil || *a.Enabled
}

// 校验:concurrency > 0、各 max_bytes > 0、timeout > 0;
// 不增加人为最大值(计划 #3)。显式配置的非法值明确报错。
func (a AssetsProbeSettings) validate() error {
	if a.ConcurrencyValue() < 1 {
		return fmt.Errorf("assets.probe.concurrency must be at least 1, got %d", a.ConcurrencyValue())
	}
	if a.ImageMaxBytesValue() < 1 {
		return fmt.Errorf("assets.probe.image_max_bytes must be positive, got %d", a.ImageMaxBytesValue())
	}
	if a.PlaylistMaxBytesValue() < 1 {
		return fmt.Errorf("assets.probe.playlist_max_bytes must be positive, got %d", a.PlaylistMaxBytesValue())
	}
	if a.VideoSegmentMaxBytesValue() < 1 {
		return fmt.Errorf("assets.probe.video_segment_max_bytes must be positive, got %d", a.VideoSegmentMaxBytesValue())
	}
	timeout, err := time.ParseDuration(a.TimeoutValue())
	if err != nil || timeout <= 0 {
		return fmt.Errorf("assets.probe.timeout %q must be a positive duration", a.TimeoutValue())
	}
	return nil
}
