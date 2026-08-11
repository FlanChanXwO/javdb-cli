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
