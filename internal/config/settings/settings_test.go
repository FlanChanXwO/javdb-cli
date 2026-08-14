package settings

import (
	"path/filepath"
	"testing"
)

func TestDefaultsUseAutoHost(t *testing.T) {
	if got := Defaults().Host; got != HostAuto {
		t.Fatalf("Defaults().Host = %q, want %q", got, HostAuto)
	}

	got, err := LoadFile(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if got.Host != HostAuto {
		t.Fatalf("LoadFile() missing host = %q, want %q", got.Host, HostAuto)
	}
	rt, err := Resolve(got, "", "", nil)
	if err != nil {
		t.Fatalf("Resolve() defaults error = %v", err)
	}
	if rt.Host != HostAuto || rt.BaseURL != "" {
		t.Fatalf("Resolve() defaults host/base = %q/%q, want %q/empty", rt.Host, rt.BaseURL, HostAuto)
	}
}

func TestResolveNormalizesEffectiveHost(t *testing.T) {
	t.Setenv("JAVDB_HOST", "")

	rt, err := Resolve(
		Settings{Host: "  HTTPS://Example.Invalid/API///  "},
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	const want = "HTTPS://Example.Invalid/API"
	if rt.Host != want || rt.BaseURL != want {
		t.Fatalf("Resolve() host/base = %q/%q, want %q", rt.Host, rt.BaseURL, want)
	}

	rt, err = Resolve(Defaults(), "  MAIN  ", "", nil)
	if err != nil {
		t.Fatalf("Resolve() logical host error = %v", err)
	}
	if rt.Host != HostMain || rt.BaseURL != HostURLs[HostMain] {
		t.Fatalf("Resolve() logical host/base = %q/%q, want %q/%q", rt.Host, rt.BaseURL, HostMain, HostURLs[HostMain])
	}
}

// TestResolveBlankEffectiveHostUsesAuto 覆盖 config/env/flag 三条入口：仅由空白组成的 host
// 与未设置 host 语义一致，统一回到 auto，不能留下 Host/BaseURL 都为空的无效运行时。
func TestResolveBlankEffectiveHostUsesAuto(t *testing.T) {
	tests := []struct {
		name     string
		fileHost string
		envHost  string
		flagHost string
	}{
		{name: "file", fileHost: "   "},
		{name: "environment", fileHost: HostMain, envHost: "   "},
		{name: "flag", fileHost: HostMain, envHost: HostMirror, flagHost: "   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JAVDB_HOST", tc.envHost)
			rt, err := Resolve(Settings{Host: tc.fileHost}, tc.flagHost, "", nil)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if rt.Host != HostAuto || rt.BaseURL != "" {
				t.Fatalf("Resolve() host/base = %q/%q, want %q/empty", rt.Host, rt.BaseURL, HostAuto)
			}
		})
	}
}

// TestResolveBlankProxyHandling 覆盖 config/env/flag 三条入口的空白 proxy 语义：config/env
// 来源的空白规范为空串（与 validator 的"空即合法"一致），显式 flag 空白则直接报错，避免
// 静默覆盖继承代理并直连绕过网络策略。
func TestResolveBlankProxyHandling(t *testing.T) {
	clearProxyEnv(t)
	t.Setenv("JAVDB_HOST", HostMirror)

	t.Run("file", func(t *testing.T) {
		rt, err := Resolve(Settings{Host: HostMirror, HTTPSProxy: "   "}, "", "", nil)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if rt.Proxy != "" {
			t.Fatalf("file proxy = %q, want empty", rt.Proxy)
		}
	})

	t.Run("environment", func(t *testing.T) {
		t.Setenv("HTTPS_PROXY", "   ")
		rt, err := Resolve(Settings{Host: HostMirror}, "", "", nil)
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if rt.Proxy != "" {
			t.Fatalf("env proxy = %q, want empty", rt.Proxy)
		}
	})

	t.Run("flag", func(t *testing.T) {
		t.Setenv("HTTPS_PROXY", "")
		if _, err := Resolve(Settings{Host: HostMirror}, "", "   ", nil); err == nil {
			t.Fatal("Resolve() accepted blank flag proxy")
		}
	})
}

func TestResolvePrecedence(t *testing.T) {
	file := Settings{Host: HostMain, HTTPSProxy: "http://file", AutoRelogin: false}
	t.Setenv("JAVDB_HOST", "mirror")
	t.Setenv("HTTPS_PROXY", "http://env")
	t.Setenv("JAVDB_AUTO_RELOGIN", "true")

	rt, err := Resolve(file, "", "", nil)
	if err != nil {
		t.Fatalf("Resolve() env error = %v", err)
	}
	if rt.Host != HostMirror {
		t.Fatalf("host env: %s", rt.Host)
	}
	if rt.Proxy != "http://env" {
		t.Fatalf("proxy env: %s", rt.Proxy)
	}
	if !rt.AutoRelogin {
		t.Fatal("auto_relogin env")
	}

	// flags win
	trueVal := false
	rt, err = Resolve(file, HostMain, "http://flag", &trueVal)
	if err != nil {
		t.Fatalf("Resolve() flags error = %v", err)
	}
	if rt.Host != HostMain || rt.Proxy != "http://flag" || rt.AutoRelogin {
		t.Fatalf("flags: %+v", rt)
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	s := Settings{Host: HostMirror, AutoRelogin: true, Lang: "zh-TW"}
	if err := SaveFile(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != HostMirror || !got.AutoRelogin || got.Lang != "zh-TW" {
		t.Fatalf("%+v", got)
	}
	// missing
	got, err = LoadFile(filepath.Join(dir, "nope.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != HostAuto {
		t.Fatalf("defaults: %+v", got)
	}
}

func TestValidateHost(t *testing.T) {
	if err := ValidateHost(""); err != nil {
		t.Fatalf("empty host: %v", err)
	}
	if err := ValidateHost("mirror"); err != nil {
		t.Fatalf("mirror: %v", err)
	}
	if err := ValidateHost("AUTO"); err != nil {
		t.Fatalf("auto: %v", err)
	}
	if err := ValidateHost("https://example.invalid"); err != nil {
		t.Fatalf("url: %v", err)
	}
	if err := ValidateHost("https://"); err == nil {
		t.Fatal("URL without authority accepted")
	}
	if err := ValidateHost("bogus"); err == nil {
		t.Fatal("bogus host accepted")
	}
}

func TestResolveRejectsInvalidEffectiveHost(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		t.Setenv("JAVDB_HOST", "")
		if _, err := Resolve(Settings{Host: "bogus"}, "", "", nil); err == nil {
			t.Fatal("Resolve() accepted invalid file host")
		}
	})

	t.Run("environment", func(t *testing.T) {
		t.Setenv("JAVDB_HOST", "bogus")
		if _, err := Resolve(Defaults(), "", "", nil); err == nil {
			t.Fatal("Resolve() accepted invalid environment host")
		}
	})

	t.Run("flag", func(t *testing.T) {
		t.Setenv("JAVDB_HOST", HostMirror)
		if _, err := Resolve(Defaults(), "bogus", "", nil); err == nil {
			t.Fatal("Resolve() accepted invalid flag host")
		}
	})
}

// TestResolveRejectsHostWithQueryOrFragment 覆盖 config(文件)/env/flag 三条入口：绝对 URL
// 带 query 或 fragment 时，transport 以字符串拼接追加 API path 会把 endpoint 拼进 query
// 或根本不发给服务器，必须拒绝。
func TestResolveRejectsHostWithQueryOrFragment(t *testing.T) {
	for _, bad := range []string{
		"https://example.invalid?a=1",
		"https://example.invalid#frag",
		"https://example.invalid?",
		"https://example.invalid#",
	} {
		t.Run("file "+bad, func(t *testing.T) {
			t.Setenv("JAVDB_HOST", "")
			if _, err := Resolve(Settings{Host: bad}, "", "", nil); err == nil {
				t.Fatalf("Resolve() accepted file host %q", bad)
			}
		})
		t.Run("environment "+bad, func(t *testing.T) {
			t.Setenv("JAVDB_HOST", bad)
			if _, err := Resolve(Defaults(), "", "", nil); err == nil {
				t.Fatalf("Resolve() accepted env host %q", bad)
			}
		})
		t.Run("flag "+bad, func(t *testing.T) {
			t.Setenv("JAVDB_HOST", "")
			if _, err := Resolve(Defaults(), bad, "", nil); err == nil {
				t.Fatalf("Resolve() accepted flag host %q", bad)
			}
		})
	}
	if err := ValidateHost("https://example.invalid?a=1"); err == nil {
		t.Fatal("ValidateHost() accepted query host")
	}
	if err := ValidateHost("https://example.invalid#frag"); err == nil {
		t.Fatal("ValidateHost() accepted fragment host")
	}
}

func TestResolveValidFlagOverridesInvalidLowerPrecedenceHosts(t *testing.T) {
	t.Setenv("JAVDB_HOST", "invalid-environment-host")
	rt, err := Resolve(Settings{Host: "invalid-file-host"}, HostMain, "", nil)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if rt.Host != HostMain || rt.BaseURL != HostURLs[HostMain] {
		t.Fatalf("Resolve() host/base = %q/%q, want %q/%q", rt.Host, rt.BaseURL, HostMain, HostURLs[HostMain])
	}
}

func clearProxyEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy", "ALL_PROXY", "all_proxy"} {
		t.Setenv(key, "")
	}
}
