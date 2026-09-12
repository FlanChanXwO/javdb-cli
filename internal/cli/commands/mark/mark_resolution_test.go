package mark

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// TestMarkMovieEnvelopeIDIsAuthoritative 验证 movie envelope 的内部 ID 优先于 ref，
// 带 ID 的输入不得再把内部 ID 当成番号搜索。
func TestMarkMovieEnvelopeIDIsAuthoritative(t *testing.T) {
	setMarkTestHome(t)
	var searchCalls, mutationCalls int
	var mutationID, mutationMethod string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "OTHER-1", "id": "wrong-id"}},
			}})
		case "/api/v1/movies/movie-internal-id/reviews":
			mutationCalls++
			mutationID = "movie-internal-id"
			mutationMethod = request.Method
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"id": "review-1",
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(
		"{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"movie-internal-id\"}\n",
	), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--watched", "--ndjson"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("search calls = %d, want 0 for authoritative envelope id", searchCalls)
	}
	if mutationCalls != 1 || mutationID != "movie-internal-id" || mutationMethod != http.MethodPost {
		t.Fatalf("mutation calls/id/method = %d/%q/%q, want 1/movie-internal-id/POST", mutationCalls, mutationID, mutationMethod)
	}
	if got := streams.Out.(*bytes.Buffer).String(); !strings.Contains(got, `"id":"movie-internal-id"`) {
		t.Fatalf("output = %q, want authoritative movie id", got)
	}
}

// TestMarkMovieEnvelopeWithoutIDResolvesNumber 锁定无 envelope ID 时仍按打印番号解析。
func TestMarkMovieEnvelopeWithoutIDResolvesNumber(t *testing.T) {
	setMarkTestHome(t)
	var searchCalls, mutationCalls int
	var searchedNumber, mutationID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchCalls++
			searchedNumber = request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-589", "id": "resolved-id"}},
			}})
		case "/api/v1/movies/resolved-id/reviews":
			mutationCalls++
			mutationID = "resolved-id"
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"id": "review-1",
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(
		"{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\"}\n",
	), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--want", "--ndjson"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 1 || searchedNumber != "SSIS-589" {
		t.Fatalf("search calls/number = %d/%q, want 1/SSIS-589", searchCalls, searchedNumber)
	}
	if mutationCalls != 1 || mutationID != "resolved-id" {
		t.Fatalf("mutation calls/id = %d/%q, want 1/resolved-id", mutationCalls, mutationID)
	}
}

// TestMarkPositionalIDStillSkipsResolution 覆盖位置参数 --id 的既有内部 ID 语义。
func TestMarkPositionalIDStillSkipsResolution(t *testing.T) {
	setMarkTestHome(t)
	var searchCalls, mutationCalls int
	var mutationID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchCalls++
			http.Error(writer, "unexpected search", http.StatusInternalServerError)
		case "/api/v1/movies/raw-id/reviews":
			mutationCalls++
			mutationID = "raw-id"
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"id": "review-1",
			}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--watched", "--id", "--ndjson", "raw-id"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("search calls = %d, want 0 for positional --id", searchCalls)
	}
	if mutationCalls != 1 || mutationID != "raw-id" {
		t.Fatalf("mutation calls/id = %d/%q, want 1/raw-id", mutationCalls, mutationID)
	}
}

func setMarkTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
}
