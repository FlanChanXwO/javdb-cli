package i18n

import (
	"reflect"
	"regexp"
	"testing"
)

func TestCatalogFormattingContract(t *testing.T) {
	all, err := catalogs()
	if err != nil {
		t.Fatal(err)
	}
	if len(all["en"]) != len(all["zh-CN"]) {
		t.Fatalf("catalog size mismatch: %d vs %d", len(all["en"]), len(all["zh-CN"]))
	}
	verbs := regexp.MustCompile(`%(?:\[[0-9]+\])?[-+# 0]*(?:\*|\d+)?(?:\.(?:\*|\d+))?[a-zA-Z%]`)
	for key, en := range all["en"] {
		zh, ok := all["zh-CN"][key]
		if !ok || en == "" || zh == "" {
			t.Errorf("missing message %s", key)
			continue
		}
		if !reflect.DeepEqual(verbs.FindAllString(en, -1), verbs.FindAllString(zh, -1)) {
			t.Errorf("format arguments differ: %s", key)
		}
	}
}

func TestSystemLanguagePrecedence(t *testing.T) {
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "zh_CN.UTF-8")
	t.Setenv("JAVDB_LANG", "en")
	if System() != "zh-CN" {
		t.Fatal("LANG was not selected independently of API language")
	}
	t.Setenv("LC_MESSAGES", "en_US.UTF-8")
	if System() != "en" {
		t.Fatal("LC_MESSAGES precedence")
	}
	t.Setenv("LC_ALL", "C")
	if System() != "en" {
		t.Fatal("C locale must be English")
	}
}
