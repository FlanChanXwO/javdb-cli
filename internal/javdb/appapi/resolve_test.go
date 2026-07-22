package appapi

import "testing"

func TestResolveNumberExactMatch(t *testing.T) {
	movies := []map[string]any{
		{"number": "SSIS-001", "id": "a"},
		{"number": "SSIS-589", "id": "9DGB5X"},
	}
	id, err := ResolveNumber(movies, "ssis-589")
	if err != nil || id != "9DGB5X" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveNumberFallbackFirst(t *testing.T) {
	movies := []map[string]any{
		{"number": "OTHER", "id": "first"},
	}
	id, err := ResolveNumber(movies, "NOPE")
	if err != nil || id != "first" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestResolveNumberEmpty(t *testing.T) {
	_, err := ResolveNumber(nil, "SSIS-589")
	if err == nil {
		t.Fatal("expected error")
	}
}
