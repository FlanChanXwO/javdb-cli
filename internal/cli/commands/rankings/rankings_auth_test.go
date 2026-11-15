package rankings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/spf13/cobra"
)

func TestRankingsAuthentication(t *testing.T) {
	for _, signedIn := range []bool{false, true} {
		name, wantAuth := "anonymous", ""
		if signedIn {
			name, wantAuth = "saved-account", "Bearer fixture-token"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				kind string
				path string
				data string
			}{
				{"movies", "/api/v1/rankings", `{"movies":[{"id":"movie-fixture","number":"FIX-001"}]}`},
				{"actors", "/api/v1/rankings/actors", `{"actors":[{"id":"actor-fixture","name":"Fixture Actor"}]}`},
				{"playback", "/api/v1/rankings/playback", `{"movies":[{"id":"playback-fixture","number":"FIX-002"}]}`},
			} {
				t.Run(tc.kind, func(t *testing.T) {
					home := t.TempDir()
					t.Setenv("HOME", home)
					t.Setenv("USERPROFILE", home)
					for _, key := range []string{"HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy", "JAVDB_HOST"} {
						t.Setenv(key, "")
					}
					dir := filepath.Join(home, ".javdb-cli")
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					fixture := []byte(`{"default_user_id":7,"accounts":[{"user_id":7,"username":"fixture","token":"fixture-token"}]}`)
					if signedIn {
						if err := os.WriteFile(filepath.Join(dir, "auth.json"), fixture, 0600); err != nil {
							t.Fatal(err)
						}
					}
					var calls atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Method != http.MethodGet || r.URL.Path != tc.path {
							t.Errorf("request = %s %s, want GET %s", r.Method, r.URL.Path, tc.path)
							http.Error(w, "unexpected rankings request", http.StatusNotFound)
							return
						}
						gotAuth := r.Header.Get("Authorization")
						w.Header().Set("Content-Type", "application/json")
						if gotAuth != wantAuth {
							t.Errorf("Authorization = %q, want %q", gotAuth, wantAuth)
							_, _ = w.Write([]byte(`{"success":0,"action":"JWTVerificationError","message":"login required"}`))
							return
						}
						_, _ = w.Write([]byte(`{"success":1,"data":` + tc.data + `}`))
					}))
					defer server.Close()
					var out, errOut bytes.Buffer
					streams := invocation.NewStreams(strings.NewReader(""), &out, &errOut)
					opts := &invocation.RootOptions{Host: server.URL}
					var cmd *cobra.Command
					switch tc.kind {
					case "movies":
						cmd = NewMovies(opts, streams)
					case "actors":
						cmd = NewActors(opts, streams)
					case "playback":
						cmd = NewPlayback(opts, streams)
					}
					cmd.SetArgs([]string{"--json"})
					if err := cmd.Execute(); err != nil {
						t.Fatalf("rankings failed: %v", err)
					}
					if got := calls.Load(); got != 1 {
						t.Fatalf("rankings requests = %d, want 1", got)
					}
					var result map[string]any
					if err := json.Unmarshal(out.Bytes(), &result); err != nil {
						t.Fatalf("invalid JSON: %v", err)
					}
					var want map[string]any
					if err := json.Unmarshal([]byte(tc.data), &want); err != nil {
						t.Fatalf("invalid fixture JSON: %v", err)
					}
					if !reflect.DeepEqual(result, want) {
						t.Fatalf("rankings output = %s, want %s", out.Bytes(), tc.data)
					}
				})
			}
		})
	}
}
