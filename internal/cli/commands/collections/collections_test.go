package collections

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

func TestNewBuildsCollectionsCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "collections" || cmd.Use != "collections KIND" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
}

func TestNewRequiresKind(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "collections: input required") {
		t.Fatalf("expected arg error, got %v", err)
	}
}

func TestNewRejectsInvalidSelectorBeforeClientSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"invalid", "--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "collection kind must be one of actors|series|codes|makers|directors") {
		t.Fatalf("invalid selector error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("invalid selector sent %d request(s), want none", requests)
	}
	for _, name := range []string{"config.toml", "device_uuid"} {
		if _, statErr := os.Stat(filepath.Join(home, ".javdb-cli", name)); !os.IsNotExist(statErr) {
			t.Fatalf("invalid selector created %s: %v", name, statErr)
		}
	}
}

func TestNewRejectsInvalidSelectorBatchBeforeClientSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("actors\ninvalid\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--ndjson"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "collection kind must be one of actors|series|codes|makers|directors") {
		t.Fatalf("invalid selector error = %v", err)
	}
	if requests != 0 {
		t.Fatalf("invalid selector sent %d request(s), want none", requests)
	}
	for _, name := range []string{"config.toml", "device_uuid"} {
		if _, statErr := os.Stat(filepath.Join(home, ".javdb-cli", name)); !os.IsNotExist(statErr) {
			t.Fatalf("invalid selector created %s: %v", name, statErr)
		}
	}
}

type collectionFixture struct {
	selector string
	path     string
	key      string
	kind     pipeline.Kind
	items    []map[string]any
}

func TestNewNDJSONFansOutCollectionSelectors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fixtures := []collectionFixture{
		{selector: "actors", path: "/api/v1/users/collected_actors", key: "actors", kind: pipeline.KindActor, items: []map[string]any{
			{"id": "actor-1", "name_zht": "演员甲", "name": "Actor One", "videos_count": float64(3)},
			{"id": "actor-2", "name": "", "videos_count": float64(1)},
		}},
		{selector: "series", path: "/api/v1/users/collected_series", key: "series", kind: pipeline.KindSeries, items: []map[string]any{{"id": "series-1", "name": "Series One"}}},
		{selector: "codes", path: "/api/v1/users/collected_codes", key: "codes", kind: pipeline.KindCode, items: []map[string]any{{"id": "code-1", "name": "Code One"}}},
		{selector: "makers", path: "/api/v1/users/collected_makers", key: "makers", kind: pipeline.KindMaker, items: []map[string]any{{"id": "maker-1", "name": "Maker One"}}},
		{selector: "directors", path: "/api/v1/users/collected_directors", key: "directors", kind: pipeline.KindDirector, items: []map[string]any{{"id": "director-1", "name": "Director One"}}},
	}
	byPath := make(map[string]collectionFixture, len(fixtures))
	for _, fixture := range fixtures {
		byPath[fixture.path] = fixture
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture, ok := byPath[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		items := fixture.items
		if r.URL.Query().Get("page") == "2" {
			items = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{fixture.key: items}})
	}))
	defer server.Close()

	for _, fixture := range fixtures {
		t.Run(fixture.selector, func(t *testing.T) {
			streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
			cmd.SetArgs([]string{fixture.selector, "--ndjson"})
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
				wantRef := row.Name
				if wantRef == "" {
					wantRef = row.ID
				}
				if envelope.Kind != fixture.kind || envelope.ID != row.ID || envelope.Ref != wantRef {
					t.Fatalf("line %d: envelope = %#v, want kind=%q id=%q ref=%q", index+1, envelope, fixture.kind, row.ID, wantRef)
				}
				entity, ok := envelope.Data["entity"].(map[string]any)
				if !ok || !reflect.DeepEqual(entity, item) {
					t.Fatalf("line %d: data.entity = %#v, want original item %#v", index+1, envelope.Data["entity"], item)
				}
			}
		})
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
			items = []map[string]any{{"id": "actor-1", "name": "Actor One", "videos_count": 2}, {"id": "actor-2", "name": "Actor Two"}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"actors": items}})
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
	if got, want := humanStreams.Out.(*bytes.Buffer).String(), "actor-1\tActor One\t2\nactor-2\tActor Two\n"; got != want {
		t.Fatalf("legacy human output = %q, want %q", got, want)
	}
}

func TestWriteNamedEmpty(t *testing.T) {
	var out, errb bytes.Buffer
	if err := writeNamed(&out, &errb, nil); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" || errb.String() != "(空列表)\n" {
		t.Fatalf("out=%q err=%q", out.String(), errb.String())
	}
}
