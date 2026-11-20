package i18n

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/width"
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
		if strings.HasSuffix(en, "\n") != strings.HasSuffix(zh, "\n") {
			t.Errorf("trailing newline differs: %s", key)
		}
	}
}

func TestAssetHeaderAlignment(t *testing.T) {
	all, err := catalogs()
	if err != nil {
		t.Fatal(err)
	}
	for locale, messages := range all {
		for _, indexWidth := range []int{1, 3} {
			header := fmt.Sprintf(messages["assets.table_header"], indexWidth, "#")
			row := fmt.Sprintf("%*d  %-5s  %-9s  %-8s  %s\n", indexWidth, 1, "image", "640x480", "1s", "preview")
			if got, want := columnStarts(header), columnStarts(row); !reflect.DeepEqual(got, want) {
				t.Errorf("%s index width %d: columns %v, want %v; header=%q", locale, indexWidth, got, want, header)
			}
		}
	}
}

func columnStarts(line string) []int {
	var starts []int
	column, inField := 0, false
	for _, r := range line {
		if unicode.IsSpace(r) {
			inField = false
		} else if !inField {
			starts = append(starts, column)
			inField = true
		}
		switch width.LookupRune(r).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			column += 2
		default:
			column++
		}
	}
	return starts
}

func TestChineseTerminology(t *testing.T) {
	all, err := catalogs()
	if err != nil {
		t.Fatal(err)
	}
	forbidden := regexp.MustCompile(`合集|播放列表|标志|厂牌|制作商|已看|磁力链($|[^接])|影片代码|榜单|envelope|信封|taxonomy|\([^\n)]*\p{Han}`)
	for key, message := range all["zh-CN"] {
		if match := forbidden.FindString(message); match != "" {
			t.Errorf("%s: forbidden wording %q in %q", key, match, message)
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
	for _, locale := range []string{"zh_TW.UTF-8", "zh_HK.UTF-8"} {
		t.Setenv("LC_ALL", locale)
		if System() != "zh-CN" {
			t.Errorf("%s must select Simplified Chinese", locale)
		}
	}
}
