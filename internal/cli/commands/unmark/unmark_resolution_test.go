package unmark

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// TestUnmarkMovieEnvelopeIDIsAuthoritative 验证 movie envelope 的内部 ID 优先于 ref，
// 带 ID 的输入不得再把内部 ID 当成番号搜索。
func TestUnmarkMovieEnvelopeIDIsAuthoritative(t *testing.T) {
	setUnmarkTestHome(t)
	var searchCalls, detailCalls, mutationCalls int
	var detailID, mutationID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "OTHER-1", "id": "wrong-id"}},
			}})
		case "/api/v4/movies/movie-internal-id":
			detailCalls++
			detailID = "movie-internal-id"
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"review": map[string]any{"id": 123}},
			}})
		case "/api/v1/movies/movie-internal-id/reviews/123":
			mutationCalls++
			mutationID = "movie-internal-id"
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(
		"{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\",\"id\":\"movie-internal-id\"}\n",
	), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("search calls = %d, want 0 for authoritative envelope id", searchCalls)
	}
	if detailCalls != 1 || detailID != "movie-internal-id" {
		t.Fatalf("detail calls/id = %d/%q, want 1/movie-internal-id", detailCalls, detailID)
	}
	if mutationCalls != 1 || mutationID != "movie-internal-id" {
		t.Fatalf("mutation calls/id = %d/%q, want 1/movie-internal-id", mutationCalls, mutationID)
	}
	if got := streams.Out.(*bytes.Buffer).String(); !strings.Contains(got, `"removed":true`) {
		t.Fatalf("output = %q, want removed=true", got)
	}
}

// TestUnmarkMovieEnvelopeWithoutIDResolvesNumber 锁定无 envelope ID 时仍按打印番号解析。
func TestUnmarkMovieEnvelopeWithoutIDResolvesNumber(t *testing.T) {
	setUnmarkTestHome(t)
	var searchCalls, detailCalls, mutationCalls int
	var searchedNumber, detailID, mutationID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchCalls++
			searchedNumber = request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movies": []map[string]any{{"number": "SSIS-589", "id": "resolved-id"}},
			}})
		case "/api/v4/movies/resolved-id":
			detailCalls++
			detailID = "resolved-id"
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"review": map[string]any{"id": 123}},
			}})
		case "/api/v1/movies/resolved-id/reviews/123":
			mutationCalls++
			mutationID = "resolved-id"
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(
		"{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"SSIS-589\"}\n",
	), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 1 || searchedNumber != "SSIS-589" {
		t.Fatalf("search calls/number = %d/%q, want 1/SSIS-589", searchCalls, searchedNumber)
	}
	if detailCalls != 1 || detailID != "resolved-id" {
		t.Fatalf("detail calls/id = %d/%q, want 1/resolved-id", detailCalls, detailID)
	}
	if mutationCalls != 1 || mutationID != "resolved-id" {
		t.Fatalf("mutation calls/id = %d/%q, want 1/resolved-id", mutationCalls, mutationID)
	}
}

// TestUnmarkPositionalIDStillSkipsResolution 覆盖位置参数 --id 的既有内部 ID 语义。
func TestUnmarkPositionalIDStillSkipsResolution(t *testing.T) {
	setUnmarkTestHome(t)
	var searchCalls, detailCalls, mutationCalls int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v2/search":
			searchCalls++
			http.Error(writer, "unexpected search", http.StatusInternalServerError)
		case "/api/v4/movies/raw-id":
			detailCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"review": map[string]any{"id": 123}},
			}})
		case "/api/v1/movies/raw-id/reviews/123":
			mutationCalls++
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--id", "--ndjson", "raw-id"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchCalls != 0 {
		t.Fatalf("search calls = %d, want 0 for positional --id", searchCalls)
	}
	if detailCalls != 1 || mutationCalls != 1 {
		t.Fatalf("detail/mutation calls = %d/%d, want 1/1", detailCalls, mutationCalls)
	}
}

func setUnmarkTestHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
}
