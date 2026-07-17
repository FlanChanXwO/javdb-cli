package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePrecedence(t *testing.T) {
	file := Settings{Host: HostMain, HTTPSProxy: "http://file", AutoRelogin: false}
	t.Setenv("JAVDB_HOST", "mirror")
	t.Setenv("HTTPS_PROXY", "http://env")
	t.Setenv("JAVDB_AUTO_RELOGIN", "true")

	rt := Resolve(file, "", "", nil)
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
	rt = Resolve(file, HostMain, "http://flag", &trueVal)
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
	if got.Host != HostMirror {
		t.Fatalf("defaults: %+v", got)
	}
	_ = os.Remove
}
