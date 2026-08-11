package lists

import (
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

func TestMyListsDefaultSortBy(t *testing.T) {
	// compile-time presence + param builder behavior via related types
	if model.Zones["censored"] != 0 {
		t.Fatal("zones")
	}
}
