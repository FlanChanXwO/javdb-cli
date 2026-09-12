package movie

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	transport "github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/search"
)

func TestMovieEndpointResolveMovieIDUsesStrictSearch(t *testing.T) {
	for _, tc := range []struct {
		name            string
		input           string
		movies          []map[string]any
		wantID          string
		wantQuery       string
		wantSearchCalls int
	}{
		{
			name:            "normalizes exact input before search",
			input:           "  ssis-589  ",
			wantID:          "id-exact",
			wantQuery:       "ssis-589",
			wantSearchCalls: 1,
			movies: []map[string]any{
				{"number": "SSIS-589", "id": "id-exact"},
			},
		},
		{
			name:            "fuzzy only results fail",
			input:           "SSIS-589",
			wantQuery:       "SSIS-589",
			wantSearchCalls: 1,
			movies: []map[string]any{
				{"number": "SSIS-58X", "id": "id-near"},
			},
		},
		{
			name:            "whitespace only input fails before search",
			input:           "   ",
			wantSearchCalls: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var searchCalls int
			var gotQuery url.Values
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/api/v2/search" {
					http.NotFound(writer, request)
					return
				}
				searchCalls++
				gotQuery = request.URL.Query()
				_ = json.NewEncoder(writer).Encode(map[string]any{
					"success": true,
					"data":    map[string]any{"movies": tc.movies},
				})
			}))
			defer server.Close()

			endpoint := newMovieEndpointForTest(t, server.URL)
			gotID, err := endpoint.ResolveMovieID(tc.input)
			if tc.wantID != "" {
				if err != nil {
					t.Fatalf("ResolveMovieID: %v", err)
				}
				if gotID != tc.wantID {
					t.Fatalf("id = %q, want %q", gotID, tc.wantID)
				}
			} else if err == nil {
				t.Fatalf("ResolveMovieID returned %q for invalid candidate set", gotID)
			}

			if searchCalls != tc.wantSearchCalls {
				t.Fatalf("search calls = %d, want %d", searchCalls, tc.wantSearchCalls)
			}
			if tc.wantSearchCalls > 0 {
				if got := gotQuery.Get("q"); got != tc.wantQuery {
					t.Errorf("q = %q, want %q", got, tc.wantQuery)
				}
				if got := gotQuery.Get("page"); got != "1" {
					t.Errorf("page = %q, want 1", got)
				}
				if got := gotQuery.Get("limit"); got != "100" {
					t.Errorf("limit = %q, want 100", got)
				}
				if _, ok := gotQuery["movie_type"]; ok {
					t.Errorf("movie_type = %q, want zone=all omission", gotQuery.Get("movie_type"))
				}
			}
		})
	}
}

func newMovieEndpointForTest(t *testing.T, host string) *MovieEndpoint {
	t.Helper()
	client, err := transport.New(transport.Options{Host: host})
	if err != nil {
		t.Fatalf("new transport client: %v", err)
	}
	return NewMovie(client, search.NewSearch(client))
}

func TestResolveNumberExactMatchesCaseInsensitive(t *testing.T) {
	movies := []map[string]any{
		{"number": "SSIS-589", "id": "id-a"},
		{"number": "HZGD-246", "id": "id-b"},
	}
	id, err := ResolveNumberExact(movies, "  ssis-589  ")
	if err != nil {
		t.Fatalf("ResolveNumberExact: %v", err)
	}
	if id != "id-a" {
		t.Errorf("id = %q", id)
	}
}

func TestResolveNumberExactRejectsZeroMultipleAndMissingID(t *testing.T) {
	for _, tc := range []struct {
		name   string
		movies []map[string]any
	}{
		{name: "no match", movies: []map[string]any{{"number": "HZGD-246", "id": "id-b"}}},
		{name: "two exact", movies: []map[string]any{
			{"number": "SSIS-589", "id": "id-a"},
			{"number": "ssis-589", "id": "id-b"},
		}},
		{name: "exact match without id", movies: []map[string]any{
			{"number": "SSIS-589"},
		}},
		{name: "exact match with empty id", movies: []map[string]any{
			{"number": "SSIS-589", "id": ""},
		}},
		{name: "empty input", movies: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			number := "SSIS-589"
			if tc.name == "empty input" {
				number = "  "
			}
			if _, err := ResolveNumberExact(tc.movies, number); err == nil {
				t.Fatal("ResolveNumberExact accepted ambiguous input")
			}
		})
	}
}

func TestResolveNumberExactDoesNotFallBackToFirstHit(t *testing.T) {
	// 严格解析不得回退到搜索首项：首项是近似命中（非完整相等）时必须失败。
	movies := []map[string]any{
		{"number": "HZGD-246", "id": "id-b"},
		{"number": "SSIS-58X", "id": "id-near"},
	}
	if _, err := ResolveNumberExact(movies, "SSIS-589"); err == nil {
		t.Fatal("ResolveNumberExact fell back to the first search hit")
	}
}

func TestResolveNumberKeepsLegacyFallback(t *testing.T) {
	// 旧 ResolveNumber 保持首项回退行为，图片链路不使用它。
	movies := []map[string]any{
		{"number": "HZGD-246", "id": "id-b"},
	}
	id, err := ResolveNumber(movies, "SSIS-589")
	if err != nil {
		t.Fatalf("ResolveNumber: %v", err)
	}
	if id != "id-b" {
		t.Errorf("legacy fallback id = %q", id)
	}
}
