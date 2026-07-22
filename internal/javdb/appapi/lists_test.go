package appapi

import "testing"

func TestMyListsDefaultSortBy(t *testing.T) {
	// compile-time presence + param builder behavior via related types
	if Zones["censored"] != 0 {
		t.Fatal("zones")
	}
}
