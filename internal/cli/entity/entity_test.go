package entity

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// TestExecuteResolvesQueriesAndMeta 用本地 httptest 覆盖 Execute 的解析、查询与 metadata 主路径。
func TestExecuteResolvesQueriesAndMeta(t *testing.T) {
	var moviesPath, detailPath, resolvePath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/actors/山手":
			// ResolveEntity 先按 id 请求详情；命中返回 actor meta。
			resolvePath = r.URL.Path
			_, _ = w.Write([]byte(`{"success":true,"data":{"actor":{"id":"act-1","name":"山手"}}}`))
		case "/api/v1/actors/act-1":
			detailPath = r.URL.Path
			_, _ = w.Write([]byte(`{"success":true,"data":{"actor":{"id":"act-1","name":"山手"}}}`))
		case "/api/v1/movies/tags":
			moviesPath = r.URL.Path
			_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[{"number":"SSIS-1","id":"m1","title":"T"}]}}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	result, err := Execute(t.Context(), client, "actor", "山手", Options{Zone: "censored", Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}
	if result.EntityID != "act-1" {
		t.Fatalf("entity id = %q", result.EntityID)
	}
	if len(result.Movies) != 1 || result.Movies[0]["number"] != "SSIS-1" {
		t.Fatalf("movies = %v", result.Movies)
	}
	if result.Entity["id"] != "act-1" {
		t.Fatalf("entity meta = %v", result.Entity)
	}
	if resolvePath != "/api/v1/actors/山手" || detailPath != "/api/v1/actors/act-1" || moviesPath != "/api/v1/movies/tags" {
		t.Fatalf("paths resolve=%q detail=%q movies=%q", resolvePath, detailPath, moviesPath)
	}
}

// TestExecuteMetadataDegradesToID 覆盖 EntityDetail 失败时 metadata 降级为 {"id": eid}。
func TestExecuteMetadataDegradesToID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/series/name1":
			_, _ = w.Write([]byte(`{"success":true,"data":{"series":{"id":"s-1"}}}`))
		case "/api/v1/series/s-1":
			// 详情失败 → Execute 的 EntityDetail fallback 应保留 eid。
			http.Error(w, `{"success":false,"action":"NotFound","message":"boom"}`, http.StatusNotFound)
		case "/api/v1/movies/tags":
			_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[]}}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	result, err := Execute(t.Context(), client, "series", "name1", Options{Zone: "censored", Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}
	if result.EntityID != "s-1" {
		t.Fatalf("entity id = %q", result.EntityID)
	}
	if id, _ := result.Entity["id"].(string); id != "s-1" {
		t.Fatalf("degraded meta = %v", result.Entity)
	}
}

// TestExecuteAllPages 覆盖 AllPages=true 时聚合全部页（页 2 空则停止）。
func TestExecuteAllPages(t *testing.T) {
	var maxPage int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/actors/n1":
			_, _ = w.Write([]byte(`{"success":true,"data":{"actor":{"id":"a1"}}}`))
		case "/api/v1/actors/a1":
			_, _ = w.Write([]byte(`{"success":true,"data":{"actor":{"id":"a1"}}}`))
		case "/api/v1/movies/tags":
			page := 0
			_, _ = fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
			if page > maxPage {
				maxPage = page
			}
			if page == 1 {
				_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[{"number":"N1","id":"m1"}]}}`))
				return
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[]}}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	result, err := Execute(t.Context(), client, "actor", "n1", Options{Zone: "censored", AllPages: true})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}
	if len(result.Movies) != 1 || result.Movies[0]["id"] != "m1" {
		t.Fatalf("movies = %v", result.Movies)
	}
	if maxPage != 2 {
		t.Fatalf("max page fetched = %d, want 2 (stop after page 2 empty)", maxPage)
	}
}

// TestExecuteHasMagnetsFilter 覆盖 HasMagnets=true 时丢弃 magnets_count==0 的行。
func TestExecuteHasMagnetsFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/series/s1":
			_, _ = w.Write([]byte(`{"success":true,"data":{"series":{"id":"s1"}}}`))
		case "/api/v1/series/x":
			_, _ = w.Write([]byte(`{"success":true,"data":{"series":{"id":"s1"}}}`))
		case "/api/v1/movies/tags":
			_, _ = w.Write([]byte(`{"success":true,"data":{"movies":[
				{"number":"A","id":"a","magnets_count":2},
				{"number":"B","id":"b","magnets_count":0}
			]}}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	result, err := Execute(t.Context(), client, "series", "x", Options{Zone: "censored", HasMagnets: true})
	if err != nil {
		t.Fatalf("Execute error = %v", err)
	}
	if len(result.Movies) != 1 || result.Movies[0]["number"] != "A" {
		t.Fatalf("filtered movies = %v", result.Movies)
	}
}

// TestExecutePropagatesResolveError 覆盖实体解析失败时错误向上传播。
func TestExecutePropagatesResolveError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// 详情 404 → 转搜索；搜索也返回空列表 → NotFound 错误。
		_, _ = w.Write([]byte(`{"success":false,"action":"NotFound","message":"nope"}`))
	}))
	defer server.Close()

	client, err := javdb.New(javdb.WithHost(server.URL))
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	if _, err := Execute(t.Context(), client, "actor", "nobody", Options{Zone: "censored"}); err == nil {
		t.Fatal("Execute expected an error for unresolvable entity")
	}
}
