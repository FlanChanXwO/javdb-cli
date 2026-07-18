package appapi

import "testing"

func TestSearchTypeListKey(t *testing.T) {
	if SearchTypeListKey("actor") != "actors" || SearchTypeListKey("list") != "lists" {
		t.Fatal("keys")
	}
}

func TestEntityLettersCoverList(t *testing.T) {
	if EntityLetters["list"] != "l" {
		t.Fatal(EntityLetters)
	}
	mask, err := BuildEntityFilter("list", "RZ8Bm", "censored", []string{"m"})
	if err != nil || mask != "0:l:RZ8Bm:m::" {
		t.Fatalf("%q %v", mask, err)
	}
}
