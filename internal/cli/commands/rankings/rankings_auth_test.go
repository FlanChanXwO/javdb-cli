package rankings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
			for _, kind := range []string{"movies", "actors", "playback"} {
				t.Run(kind, func(t *testing.T) {
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
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						gotAuth := r.Header.Get("Authorization")
						w.Header().Set("Content-Type", "application/json")
						if gotAuth != wantAuth {
							_, _ = w.Write([]byte(`{"success":0,"action":"JWTVerificationError","message":"login required"}`))
							return
						}
						_, _ = w.Write([]byte(`{"success":1,"data":{"movies":[],"actors":[]}}`))
					}))
					defer server.Close()
					var out, errOut bytes.Buffer
					streams := invocation.NewStreams(strings.NewReader(""), &out, &errOut)
					opts := &invocation.RootOptions{Host: server.URL}
					var cmd *cobra.Command
					switch kind {
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
					var result map[string]any
					if err := json.Unmarshal(out.Bytes(), &result); err != nil {
						t.Fatalf("invalid JSON: %v", err)
					}
					key := "movies"
					if kind == "actors" {
						key = "actors"
					}
					if _, ok := result[key]; !ok {
						t.Fatalf("missing %s output", key)
					}
				})
			}
		})
	}
}
