package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
)

// isolateHome 把 HOME 指向临时目录，避免配置命令测试污染真实本机状态。
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	return home
}

// executeConfig 用给定参数运行独立 config 命令树（无根 hook），返回输出缓冲。
func executeConfig(t *testing.T, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	t.Helper()
	var out, errb bytes.Buffer
	streams := invocation.NewStreams(strings.NewReader(""), &out, &errb)
	streams.InIsTerminal = true
	command := New(streams)
	command.SetOut(&out)
	command.SetErr(&errb)
	command.SetArgs(args)
	return out, errb, command.Execute()
}

func TestConfigUnsetOnMissingConfigIsNoOp(t *testing.T) {
	home := isolateHome(t)
	out, errb, err := executeConfig(t, "unset", "host")
	if err != nil {
		t.Fatalf("unset on missing config error = %v", err)
	}
	if out.String() != "" || errb.String() != "" {
		t.Fatalf("unset on missing config produced output: stdout=%q stderr=%q", out.String(), errb.String())
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unset on missing config created file: %v", err)
	}
}

func TestConfigUnsetOnExistingFileResetsHostToAuto(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "host", "main"); err != nil {
		t.Fatalf("set host error = %v", err)
	}
	if _, _, err := executeConfig(t, "unset", "host"); err != nil {
		t.Fatalf("unset host error = %v", err)
	}
	out, _, err := executeConfig(t, "get", "host")
	if err != nil {
		t.Fatalf("get host error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "auto" {
		t.Fatalf("host after unset = %q, want %q", got, "auto")
	}
}

func TestConfigSetCreatesPrivateFile(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "set", "lang", "zh"); err != nil {
		t.Fatalf("set lang error = %v", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	out, _, err := executeConfig(t, "get", "lang")
	if err != nil {
		t.Fatalf("get lang error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "zh" {
		t.Fatalf("lang after set = %q, want %q", got, "zh")
	}
}

func TestConfigUnsetUnknownKeyErrorsOnMissingConfig(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "unset", "bogus"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("unset bogus error = %v, want unknown key", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("unset bogus created config: %v", statErr)
	}
}

func TestConfigGetUnknownKeyErrorsWithoutCreating(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "get", "bogus"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("get bogus error = %v, want unknown key", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("get bogus created config: %v", statErr)
	}
}

func TestConfigSetUnknownKeyErrorsWithoutCreating(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "set", "bogus", "x"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("set bogus error = %v, want unknown key", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("set bogus created config: %v", statErr)
	}
}

func TestConfigSetInvalidHostValueErrorsWithoutCreating(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "set", "host", "bogus"); err == nil || !strings.Contains(err.Error(), "host must be") {
		t.Fatalf("set host bogus error = %v, want host validation error", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("set host bogus created config: %v", statErr)
	}
}

// TestConfigAssetProbeContract 在一条真实 CLI 链路上验证 [assets.probe] 的
// 有效配置契约：默认值 → set → get → unset 回默认 → 非法值被拒绝，
// 以及损坏的手写配置不得被 config get 静默隐藏。
func TestConfigAssetProbeContract(t *testing.T) {
	home := isolateHome(t)
	get := func(key string) string {
		t.Helper()
		out, _, err := executeConfig(t, "get", key)
		if err != nil {
			t.Fatalf("get %s: %v", key, err)
		}
		return strings.TrimSpace(out.String())
	}

	if got := get("assets.probe.enabled"); got != "true" {
		t.Fatalf("default enabled = %q, want true", got)
	}
	if got := get("assets.probe.concurrency"); got != "4" {
		t.Fatalf("default concurrency = %q, want 4", got)
	}

	path := filepath.Join(home, ".javdb-cli", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read default config: %v", err)
	}
	if strings.Contains(string(data), "[assets.probe]") {
		t.Fatalf("default config unexpectedly contains [assets.probe]:\n%s", data)
	}

	if _, _, err := executeConfig(t, "set", "assets.probe.concurrency", "8"); err != nil {
		t.Fatalf("set concurrency: %v", err)
	}
	if got := get("assets.probe.concurrency"); got != "8" {
		t.Fatalf("explicit concurrency = %q, want 8", got)
	}
	if _, _, err := executeConfig(t, "unset", "assets.probe.concurrency"); err != nil {
		t.Fatalf("unset concurrency: %v", err)
	}
	if got := get("assets.probe.concurrency"); got != "4" {
		t.Fatalf("reset concurrency = %q, want 4", got)
	}

	if _, _, err := executeConfig(t, "set", "assets.probe.enabled", "false"); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if got := get("assets.probe.enabled"); got != "false" {
		t.Fatalf("explicit enabled = %q, want false", got)
	}
	// 半量配置（只写 enabled）必须继续返回默认 concurrency。
	if got := get("assets.probe.concurrency"); got != "4" {
		t.Fatalf("sparse concurrency = %q, want default 4", got)
	}

	// 非正值是非法 interface 输入，必须在命令层被拒绝。
	if _, _, err := executeConfig(t, "set", "assets.probe.concurrency", "0"); err == nil {
		t.Fatal("set concurrency accepted 0")
	}

	// 手工写入损坏配置后，无参数 config get 必须报错而不是静默跳过该项。
	if err := os.WriteFile(path, []byte("[assets.probe]\nconcurrency = 0\n"), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}
	_, _, err = executeConfig(t, "get")
	if err == nil || !strings.Contains(err.Error(), "assets.probe.concurrency must be positive") {
		t.Fatalf("config get error = %v, want invalid concurrency", err)
	}
}

func TestConfigReverseSearchScalarRoundTrip(t *testing.T) {
	isolateHome(t)
	out, _, err := executeConfig(t, "set", "reverse_search.retries", "5")
	if err != nil {
		t.Fatalf("set retries: %v", err)
	}
	_ = out
	out, _, err = executeConfig(t, "get", "reverse_search.retries")
	if err != nil {
		t.Fatalf("get retries: %v", err)
	}
	if strings.TrimSpace(out.String()) != "5" {
		t.Errorf("get retries = %q, want 5", out.String())
	}

	if _, _, err := executeConfig(t, "set", "reverse_search.cache", "false"); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	out, _, err = executeConfig(t, "get", "reverse_search.cache")
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if strings.TrimSpace(out.String()) != "false" {
		t.Errorf("get cache = %q, want false", out.String())
	}

	if _, _, err := executeConfig(t, "set", "reverse_search.default_source", "custom"); err != nil {
		t.Fatalf("set default_source: %v", err)
	}
	out, _, err = executeConfig(t, "get", "reverse_search.default_source")
	if err != nil {
		t.Fatalf("get default_source: %v", err)
	}
	if strings.TrimSpace(out.String()) != "custom" {
		t.Errorf("get default_source = %q", out.String())
	}
}

func TestConfigReverseSearchUnsetFallsBackToDefaults(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "reverse_search.default_source", "custom"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := executeConfig(t, "unset", "reverse_search.default_source"); err != nil {
		t.Fatal(err)
	}
	out, _, err := executeConfig(t, "get", "reverse_search.default_source")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "builtin" {
		t.Errorf("default_source after unset = %q, want builtin", out.String())
	}
}

func TestConfigReverseSearchKeyValidation(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "reverse_search.retries", "abc"); err == nil {
		t.Fatal("set retries accepted a non-integer")
	}
	if _, _, err := executeConfig(t, "set", "reverse_search.cache", "maybe"); err == nil {
		t.Fatal("set cache accepted a non-boolean")
	}
	if _, _, err := executeConfig(t, "get", "reverse_search.unknown_key"); err == nil {
		t.Fatal("get accepted an unknown reverse_search key")
	}
	if _, _, err := executeConfig(t, "set", "reverse_search.sources", "[]"); err == nil {
		t.Fatal("set accepted the hand-edited sources array key")
	}
}

func TestConfigGetListsReverseSearchScalars(t *testing.T) {
	isolateHome(t)
	out, _, err := executeConfig(t, "get")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, line := range []string{
		"reverse_search.default_source=builtin",
		"reverse_search.cache=true",
		"reverse_search.cache_ttl=720h",
		"reverse_search.retries=3",
		"reverse_search.retry_wait=30s",
		"reverse_search.request_timeout=60s",
	} {
		if !strings.Contains(out.String(), line) {
			t.Errorf("config get output lacks %q:\n%s", line, out.String())
		}
	}
}

func assertConfigDisplayEnvelopes(t *testing.T, envelopes []pipeline.Envelope) {
	t.Helper()
	if len(envelopes) != len(displayConfigKeys) {
		t.Fatalf("envelopes = %d, want %d", len(envelopes), len(displayConfigKeys))
	}
	for index, key := range displayConfigKeys {
		envelope := envelopes[index]
		if err := envelope.Validate(); err != nil {
			t.Errorf("envelope %d (%s) invalid: %v", index, key, err)
		}
		if envelope.Kind != pipeline.KindConfigKey {
			t.Errorf("envelope %d kind = %q, want %q", index, envelope.Kind, pipeline.KindConfigKey)
		}
		if envelope.Ref != key {
			t.Errorf("envelope %d ref = %q, want %q", index, envelope.Ref, key)
		}
		value, ok := envelope.Data["value"].(string)
		if !ok {
			t.Errorf("envelope %d value = %#v, want string", index, envelope.Data["value"])
			continue
		}
		if key == "https_proxy" && value != "***" {
			t.Errorf("https_proxy value = %q, want redacted value", value)
		}
	}
}

func TestConfigGetTTYMachineModesListDisplayKeysAsEnvelopes(t *testing.T) {
	for _, mode := range []string{"--json", "--ndjson"} {
		t.Run(mode[2:], func(t *testing.T) {
			isolateHome(t)
			if _, _, err := executeConfig(t, "set", "https_proxy", "http://user:secret@proxy.example:8080"); err != nil {
				t.Fatal(err)
			}
			var out, errb bytes.Buffer
			streams := invocation.NewStreams(strings.NewReader(""), &out, &errb)
			streams.InIsTerminal = true
			command := New(streams)
			command.SetArgs([]string{"get", mode})
			if err := command.Execute(); err != nil {
				t.Fatalf("get %s: %v", mode, err)
			}
			if strings.Contains(out.String(), "secret") {
				t.Fatalf("%s output leaks proxy credentials: %s", mode, out.String())
			}

			var envelopes []pipeline.Envelope
			if mode == "--json" {
				if err := json.Unmarshal(out.Bytes(), &envelopes); err != nil {
					t.Fatalf("decode JSON output: %v\n%s", err, out.String())
				}
			} else {
				lines := strings.Split(strings.TrimSpace(out.String()), "\n")
				for index, line := range lines {
					envelope, err := pipeline.DecodeNDJSON(line)
					if err != nil {
						t.Fatalf("decode NDJSON line %d: %v\n%s", index+1, err, out.String())
					}
					envelopes = append(envelopes, envelope)
				}
			}
			assertConfigDisplayEnvelopes(t, envelopes)
		})
	}
}

func TestConfigKeyEnvelopeUsesProvidedSnapshot(t *testing.T) {
	cfg := settings.Defaults()
	cfg.Host = "snapshot-host"
	cfg.HTTPSProxy = "http://user:secret@proxy.example:8080"

	host, err := configKeyEnvelope(cfg, "host")
	if err != nil {
		t.Fatalf("host envelope: %v", err)
	}
	if got := host.Data["value"]; got != "snapshot-host" {
		t.Fatalf("host value = %#v, want snapshot-host", got)
	}

	proxy, err := configKeyEnvelope(cfg, "https_proxy")
	if err != nil {
		t.Fatalf("proxy envelope: %v", err)
	}
	if got := proxy.Data["value"]; got != "***" {
		t.Fatalf("proxy value = %#v, want redacted value", got)
	}
}

func TestConfigGetTTYRejectsJSONAndNDJSONTogether(t *testing.T) {
	isolateHome(t)
	_, _, err := executeConfig(t, "get", "--json", "--ndjson")
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("get with mutually exclusive output flags error = %v", err)
	}
}

// TestConfigGetStdinBatch 无 key + 非 TTY stdin：显式 NDJSON 批处理输出 config_key 信封。
func TestConfigGetStdinBatch(t *testing.T) {
	isolateHome(t)
	streams := invocation.NewStreams(strings.NewReader("host\nlang\n"), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(streams)
	command.SetArgs([]string{"get", "--ndjson"})
	if err := command.Execute(); err != nil {
		t.Fatalf("get batch: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], `"kind":"config_key"`) || !strings.Contains(lines[0], `"ref":"host"`) {
		t.Errorf("first envelope = %s", lines[0])
	}
	if !strings.Contains(lines[0], `"value":"auto"`) {
		t.Errorf("first envelope lacks value: %s", lines[0])
	}
}

// TestConfigGetBatchRedactsProxyCredentials 管道信封不得泄漏 proxy 凭据。
func TestConfigGetBatchRedactsProxyCredentials(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "https_proxy", "http://user:secret@proxy.example:8080"); err != nil {
		t.Fatal(err)
	}
	streams := invocation.NewStreams(strings.NewReader("https_proxy\n"), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(streams)
	command.SetArgs([]string{"get", "--ndjson"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	if strings.Contains(out, "secret") {
		t.Fatalf("pipeline envelope leaks proxy credentials: %s", out)
	}
	if !strings.Contains(out, `"value":"***"`) {
		t.Errorf("proxy value should be redacted: %s", out)
	}
}

// TestConfigUnsetStdinBatch 非 TTY stdin key 批处理 unset。
func TestConfigUnsetStdinBatch(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "lang", "zh-CN"); err != nil {
		t.Fatal(err)
	}
	streams := invocation.NewStreams(strings.NewReader("lang\n"), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(streams)
	command.SetArgs([]string{"unset"})
	if err := command.Execute(); err != nil {
		t.Fatalf("unset batch: %v", err)
	}
	out, _, err := executeConfig(t, "get", "lang")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "en" {
		t.Errorf("lang after batch unset = %q, want en", out.String())
	}
}
