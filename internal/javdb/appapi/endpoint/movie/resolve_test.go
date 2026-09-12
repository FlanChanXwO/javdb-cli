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
		name      string
		input     string
		movies    []map[string]any
		wantID    string
		wantQuery string
	}{
		{
			name:      "normalizes exact input before search",
			input:     "  ssis-589  ",
			wantID:    "id-exact",
			wantQuery: "ssis-589",
			movies: []map[string]any{
				{"number": "SSIS-589", "id": "id-exact"},
			},
		},
		{
			name:      "fuzzy only results fail",
			input:     "SSIS-589",
			wantQuery: "SSIS-589",
			movies: []map[string]any{
				{"number": "SSIS-58X", "id": "id-near"},
			},
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

			if searchCalls != 1 {
				t.Fatalf("search calls = %d, want 1", searchCalls)
			}
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

func TestResolveNumberExact(t *testing.T) {
	for _, tc := range []struct {
		name    string
		input   string
		movies  []map[string]any
		wantID  string
		wantErr bool
	}{
		{
			name:  "exact match is case insensitive and trims input",
			input: "  ssis-589  ",
			movies: []map[string]any{
				{"number": "SSIS-589", "id": "id-a"},
			},
			wantID: "id-a",
		},
		{
			name:  "duplicate exact rows with same id are accepted",
			input: "SSIS-589",
			movies: []map[string]any{
				{"number": "SSIS-589", "id": "id-a"},
				{"number": "ssis-589", "id": "id-a"},
			},
			wantID: "id-a",
		},
		{
			name:  "duplicate exact rows with different ids are ambiguous",
			input: "SSIS-589",
			movies: []map[string]any{
				{"number": "SSIS-589", "id": "id-a"},
				{"number": "ssis-589", "id": "id-b"},
			},
			wantErr: true,
		},
		{
			name:  "fuzzy only rows have no exact match",
			input: "SSIS-589",
			movies: []map[string]any{
				{"number": "SSIS-58X", "id": "id-near"},
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, err := ResolveNumberExact(tc.movies, tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ResolveNumberExact returned %q, want error", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveNumberExact: %v", err)
			}
			if id != tc.wantID {
				t.Errorf("id = %q, want %q", id, tc.wantID)
			}
		})
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
