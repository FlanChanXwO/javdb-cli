package javdb_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

func TestClientResolveMovieIDUsesStrictEndpointResolution(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  string
		movies []map[string]any
		wantID string
	}{
		{
			name:  "exact match accepts case and surrounding whitespace",
			input: "  SSIS-589  ",
			movies: []map[string]any{
				{"number": "ssis-589", "id": "id-exact"},
			},
			wantID: "id-exact",
		},
		{
			name:  "fuzzy-only result returns an error",
			input: "SSIS-589",
			movies: []map[string]any{
				{"number": "SSIS-58X", "id": "id-near"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != "/api/v2/search" {
					http.NotFound(writer, request)
					return
				}
				_ = json.NewEncoder(writer).Encode(map[string]any{
					"success": true,
					"data":    map[string]any{"movies": tc.movies},
				})
			}))
			defer server.Close()

			client, err := javdb.New(javdb.WithHost(server.URL))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			gotID, err := client.ResolveMovieID(context.Background(), tc.input)
			if tc.wantID != "" {
				if err != nil {
					t.Fatalf("ResolveMovieID: %v", err)
				}
				if gotID != tc.wantID {
					t.Fatalf("id = %q, want %q", gotID, tc.wantID)
				}
			} else if err == nil {
				t.Fatalf("ResolveMovieID returned %q for fuzzy-only result", gotID)
			}
		})
	}
}
