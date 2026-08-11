package entity

import (
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/browse"
)

func TestSearchTypeListKey(t *testing.T) {
	if SearchTypeListKey("actor") != "actors" || SearchTypeListKey("list") != "lists" {
		t.Fatal("keys")
	}
}

func TestEntityLettersCoverList(t *testing.T) {
	if browse.EntityLetters["list"] != "l" {
		t.Fatal(browse.EntityLetters)
	}
	mask, err := browse.BuildEntityFilter("list", "RZ8Bm", "censored", []string{"m"})
	if err != nil || mask != "0:l:RZ8Bm:m::" {
		t.Fatalf("%q %v", mask, err)
	}
}
