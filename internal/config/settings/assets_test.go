package settings

import (
	"os"
	"path/filepath"
	"testing"
)

// assets.probe 配置契约(计划 #3):
// 整表不存在等效默认;部分字段配置其余默认;校验 positive 约束。

func TestAssetsProbeDefaults(t *testing.T) {
	probe := AssetsProbeSettings{}
	probe.applyDefaults()
	if !probe.EnabledValue() {
		t.Fatal("probe default must be enabled")
	}
	if probe.ConcurrencyValue() != 4 {
		t.Fatalf("concurrency = %d, want 4", probe.Concurrency)
	}
	if probe.ImageMaxBytesValue() != 65536 {
		t.Fatalf("image_max_bytes = %d, want 65536", probe.ImageMaxBytes)
	}
	if probe.PlaylistMaxBytesValue() != 262144 {
		t.Fatalf("playlist_max_bytes = %d, want 262144", probe.PlaylistMaxBytes)
	}
	if probe.VideoSegmentMaxBytesValue() != 262144 {
		t.Fatalf("video_segment_max_bytes = %d, want 262144", probe.VideoSegmentMaxBytes)
	}
	if probe.TimeoutValue() != "10s" {
		t.Fatalf("timeout = %q, want 10s", probe.TimeoutValue())
	}
	if err := probe.validate(); err != nil {
		t.Fatalf("defaults must validate: %v", err)
	}
}

// 部分字段配置,其余继续默认。
func TestAssetsProbePartialConfigKeepsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[assets.probe]\nconcurrency = 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Assets.Probe.ConcurrencyValue() != 2 {
		t.Fatalf("concurrency = %d, want 2", s.Assets.Probe.Concurrency)
	}
	if s.Assets.Probe.ImageMaxBytesValue() != 65536 {
		t.Fatalf("image_max_bytes = %d, want default 65536", s.Assets.Probe.ImageMaxBytes)
	}
	if !s.Assets.Probe.EnabledValue() {
		t.Fatal("enabled must default to true")
	}
}

// disabled=false 完全关闭 probe。
func TestAssetsProbeDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[assets.probe]\nenabled = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Assets.Probe.EnabledValue() {
		t.Fatal("enabled=false must disable probe")
	}
}

// 整表不存在:等效默认配置。
func TestAssetsProbeAbsentTableIsDefaults(t *testing.T) {
	s, err := LoadFile(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Assets.Probe.ConcurrencyValue() != 4 || s.Assets.Probe.ImageMaxBytesValue() != 65536 {
		t.Fatalf("absent table = %+v, want defaults", s.Assets.Probe)
	}
}

// 校验:非法值明确报错。
func TestAssetsProbeValidateRejectsInvalid(t *testing.T) {
	cases := []struct {
		name   string
		config string
	}{
		{"zero concurrency", "[assets.probe]\nconcurrency = 0\n"},
		{"zero image max bytes", "[assets.probe]\nimage_max_bytes = 0\n"},
		{"invalid timeout", "[assets.probe]\ntimeout = \"abc\"\n"},
		{"negative timeout", "[assets.probe]\ntimeout = \"-5s\"\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(tc.config), 0o600); err != nil {
				t.Fatal(err)
			}
			s, err := LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Assets.Probe.validate(); err == nil {
				t.Fatalf("invalid config must be rejected: %+v", s.Assets.Probe)
			}
		})
	}
}
