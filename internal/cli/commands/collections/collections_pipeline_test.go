package collections

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

type collectionFixture struct {
	path  string
	key   string
	kind  pipeline.Kind
	items []map[string]any
}

func TestNewNDJSONFansOutAllCollectionSelectors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fixtures := map[string]collectionFixture{
		"actors": {
			path: "/api/v1/users/collected_actors", key: "actors", kind: pipeline.KindActor,
			items: []map[string]any{
				{"id": "actor-1", "name_zht": "演员甲", "name": "Actor One", "videos_count": 3},
				{"id": "actor-2", "name": "", "videos_count": 1},
			},
		},
		"series": {
			path: "/api/v1/users/collected_series", key: "series", kind: pipeline.KindSeries,
			items: []map[string]any{{"id": "series-1", "name": "Series One"}},
		},
		"codes": {
			path: "/api/v1/users/collected_codes", key: "codes", kind: pipeline.KindCode,
			items: []map[string]any{{"id": "code-1", "name": "Code One"}},
		},
		"makers": {
			path: "/api/v1/users/collected_makers", key: "makers", kind: pipeline.KindMaker,
			items: []map[string]any{{"id": "maker-1", "name": "Maker One"}},
		},
		"directors": {
			path: "/api/v1/users/collected_directors", key: "directors", kind: pipeline.KindDirector,
			items: []map[string]any{{"id": "director-1", "name": "Director One"}},
		},
	}
	pathToFixture := make(map[string]collectionFixture, len(fixtures))
	for _, fixture := range fixtures {
		pathToFixture[fixture.path] = fixture
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture, ok := pathToFixture[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		var items []map[string]any
		switch r.URL.Query().Get("page") {
		case "1":
			items = fixture.items
		case "2":
			items = []map[string]any{}
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{fixture.key: items},
		})
	}))
	defer server.Close()

	for selector, fixture := range fixtures {
		t.Run(selector, func(t *testing.T) {
			streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
			cmd.SetArgs([]string{selector, "--ndjson"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("NDJSON execute: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(streams.Out.(*bytes.Buffer).String()), "\n")
			if len(lines) != len(fixture.items) {
				t.Fatalf("NDJSON lines = %d, want %d: %q", len(lines), len(fixture.items), streams.Out.(*bytes.Buffer).String())
			}
			for index, line := range lines {
				envelope, err := pipeline.DecodeNDJSON(line)
				if err != nil {
					t.Fatalf("line %d: decode envelope: %v", index+1, err)
				}
				item := fixture.items[index]
				row := result.ProjectNamed(item)
				ref := row.Name
				if ref == "" {
					ref = row.ID
				}
				if envelope.Kind != fixture.kind || envelope.ID != row.ID || envelope.Ref != ref {
					t.Fatalf("line %d: envelope = %#v, want kind=%q id=%q ref=%q", index+1, envelope, fixture.kind, row.ID, ref)
				}
				entity, ok := envelope.Data["entity"].(map[string]any)
				if !ok || entity["id"] != row.ID {
					t.Fatalf("line %d: data.entity = %#v, want original item", index+1, envelope.Data["entity"])
				}
			}
		})
	}
}

func TestNewNDJSONRejectsEntityWithoutNameOrID(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/collected_actors" {
			http.NotFound(w, r)
			return
		}
		items := []map[string]any{}
		if r.URL.Query().Get("page") == "1" {
			items = []map[string]any{{"id": "", "name": ""}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"actors": items},
		})
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"actors", "--ndjson"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "collections completed with 1 of 1 items failed") {
		t.Fatalf("nameless entity error = %v", err)
	}
	envelope, err := pipeline.DecodeNDJSON(strings.TrimSpace(streams.Out.(*bytes.Buffer).String()))
	if err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Kind != pipeline.KindError || envelope.Data["code"] != "item" {
		t.Fatalf("error envelope = %#v", envelope)
	}
}

func TestNewLegacyJSONAndHumanOutputRemainAggregate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	fs, store, err := authstore.Open()
	if err != nil {
		t.Fatalf("open auth store: %v", err)
	}
	store.Upsert(auth.Account{UserID: 1, Username: "test", Token: "token"}, true)
	if err := fs.Commit(store); err != nil {
		t.Fatalf("commit auth store: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/users/collected_actors" {
			http.NotFound(w, r)
			return
		}
		items := []map[string]any{}
		if r.URL.Query().Get("page") == "1" {
			items = []map[string]any{
				{"id": "actor-1", "name": "Actor One", "videos_count": 2},
				{"id": "actor-2", "name": "Actor Two"},
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data":    map[string]any{"actors": items},
		})
	}))
	defer server.Close()

	jsonStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	jsonCmd := New(&invocation.RootOptions{Host: server.URL}, jsonStreams)
	jsonCmd.SetArgs([]string{"actors", "--json"})
	if err := jsonCmd.Execute(); err != nil {
		t.Fatalf("legacy JSON execute: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(jsonStreams.Out.(*bytes.Buffer).Bytes(), &got); err != nil {
		t.Fatalf("legacy JSON output = %q: %v", jsonStreams.Out.(*bytes.Buffer).String(), err)
	}
	items, ok := got["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("legacy JSON items = %#v, want two items", got["items"])
	}

	humanStreams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	humanStreams.OutIsTerminal = true
	humanCmd := New(&invocation.RootOptions{Host: server.URL}, humanStreams)
	humanCmd.SetArgs([]string{"actors"})
	if err := humanCmd.Execute(); err != nil {
		t.Fatalf("legacy human execute: %v", err)
	}
	wantHuman := "actor-1\tActor One\t2\nactor-2\tActor Two\n"
	if got := humanStreams.Out.(*bytes.Buffer).String(); got != wantHuman {
		t.Fatalf("legacy human output = %q, want %q", got, wantHuman)
	}
}
