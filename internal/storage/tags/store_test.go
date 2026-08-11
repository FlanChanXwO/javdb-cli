package tags

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAliasMapAndResolve(t *testing.T) {
	doc := &Doc{Zone: "censored", Type: 0, Categories: []Category{
		{ID: "body", NameEN: "Body", Tags: []Tag{
			{ID: "17", NameEN: "Big Tits", NameZH: "巨乳"},
		}},
	}}
	m := AliasMap(doc)
	if m["17"] != "17" || m["big tits"] != "17" || m["巨乳"] != "17" {
		t.Fatalf("%v", m)
	}
	ids, err := ResolveRefs([]string{"巨乳", "17"}, m)
	if err != nil || len(ids) != 2 || ids[0] != "17" {
		t.Fatalf("%v %v", ids, err)
	}
	_, err = ResolveRefs([]string{"nope"}, m)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tags-censored.json")
	doc := &Doc{Zone: "censored", Type: 0, Categories: []Category{
		{ID: "1", NameEN: "A", Tags: []Tag{{ID: "17", NameEN: "Big Tits", NameZH: "巨乳"}}},
	}}
	if err := Save(path, doc); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got == nil || len(got.Categories) != 1 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestLoadMissingAndMalformed(t *testing.T) {
	dir := t.TempDir()
	missing, err := Load(filepath.Join(dir, "missing.json"))
	if err != nil || missing != nil {
		t.Fatalf("missing taxonomy = %+v, %v", missing, err)
	}
	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(invalidPath); err == nil {
		t.Fatal("invalid taxonomy JSON unexpectedly loaded")
	}
	validWithoutCategories := filepath.Join(dir, "no-categories.json")
	if err := os.WriteFile(validWithoutCategories, []byte(`{"zone":"censored","type":0}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Load(validWithoutCategories)
	if err != nil || got != nil {
		t.Fatalf("taxonomy without categories = %+v, %v", got, err)
	}
}
