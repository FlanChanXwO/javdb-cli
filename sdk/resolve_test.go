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
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v2/search" {
			http.NotFound(writer, request)
			return
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"success": true,
			"data": map[string]any{"movies": []map[string]any{
				{"number": "SSIS-58X", "id": "id-near"},
			}},
		})
	}))
	defer server.Close()

	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := client.ResolveMovieID(context.Background(), "SSIS-589"); err == nil {
		t.Fatal("ResolveMovieID accepted a fuzzy-only result")
	}
}
