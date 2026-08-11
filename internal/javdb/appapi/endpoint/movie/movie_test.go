package movie

import (
	"encoding/json"
	"testing"
)

func TestMovieCommentsParamsStayOnOneRequestedPage(t *testing.T) {
	params := movieCommentsParams(3, 7)
	if params["page"] != "3" || params["limit"] != "7" {
		t.Fatalf("movie comments params = %v", params)
	}

	defaults := movieCommentsParams(0, 0)
	if defaults["page"] != "1" || defaults["limit"] != "20" {
		t.Fatalf("movie comments defaults = %v", defaults)
	}
}

func TestDecodeMovieCommentsDropsNullEntries(t *testing.T) {
	items := decodeMovieComments(json.RawMessage(`[{"id":"one"},null,{"id":"two"}]`))
	if len(items) != 2 || items[0]["id"] != "one" || items[1]["id"] != "two" {
		t.Fatalf("decoded comments = %#v", items)
	}
}
