package search

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

func TestSearchAuthentication(t *testing.T) {
	for _, signedIn := range []bool{false, true} {
		name, wantAuth := "anonymous", ""
		if signedIn {
			name, wantAuth = "saved-account", "Bearer fixture-token"
		}
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name           string
				image, magnets bool
				format         string
			}{
				{"text-json", false, false, "--json"}, {"text-ndjson", false, false, "--ndjson"},
				{"magnets", false, true, "--json"}, {"image-json", true, false, "--json"},
				{"image-ndjson", true, false, "--ndjson"}, {"image-text", true, false, ""},
				{"image-magnets", true, true, "--json"},
				{"image-url", true, false, "--json"},
			} {
				t.Run(tc.name, func(t *testing.T) {
					provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Header.Get("Authorization") != "" {
							t.Error("account token leaked to image provider")
						}
						if r.URL.Path == "/frame.jpg" {
							_, _ = w.Write(testJPEG)
							return
						}
						_, _ = w.Write([]byte(`{"results":[{"video_code":"ABC-123","best_similarity":95,"frames":[]}]}`))
					}))
					defer provider.Close()
					writeReverseSearchConfig(t, provider.URL)
					home := os.Getenv("HOME")
					for _, key := range []string{"HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy", "JAVDB_HOST"} {
						t.Setenv(key, "")
					}
					dir := filepath.Join(home, ".javdb-cli")
					var searches, details, magnets atomic.Int32

					if signedIn {
						if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"default_user_id":7,"accounts":[{"user_id":7,"username":"fixture","token":"fixture-token"}]}`), 0600); err != nil {
							t.Fatal(err)
						}
					}
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						if r.Header.Get("Authorization") != wantAuth {
							_, _ = w.Write([]byte(`{"success":0,"action":"JWTVerificationError","message":"missing token"}`))
							return
						}
						switch {
						case r.URL.Path == "/api/v2/search":
							searches.Add(1)
							_, _ = w.Write([]byte(`{"success":1,"data":{"movies":[{"id":"fixture","number":"ABC-123","title":"fixture"}]}}`))
						case strings.Contains(r.URL.Path, "magnets"):
							magnets.Add(1)
							_, _ = w.Write([]byte(`{"success":1,"data":{"magnets":[]}}`))
						case strings.HasPrefix(r.URL.Path, "/api/v4/movies/"):
							details.Add(1)
							_, _ = w.Write([]byte(`{"success":1,"data":{"movie":{"id":"fixture","number":"ABC-123","title":"fixture"}}}`))
						default:
							http.NotFound(w, r)
						}
					}))
					defer server.Close()
					var out, errOut bytes.Buffer
					cmd := New(&invocation.RootOptions{Host: server.URL}, invocation.NewStreams(strings.NewReader(""), &out, &errOut))
					input := "ABC-123"
					if tc.image {
						input = filepath.Join(t.TempDir(), "frame.jpg")
						if err := os.WriteFile(input, testJPEG, 0600); err != nil {
							t.Fatal(err)
						}
					}
					if tc.name == "image-url" {
						input = provider.URL + "/frame.jpg"
					}
					args := []string{input}
					if tc.format != "" {
						args = append(args, tc.format)
					}
					if tc.magnets {
						args = append(args, "--magnets", "0")
					}
					if tc.image {
						args = append(args, "--no-cache")
					}
					cmd.SetArgs(args)
					if err := cmd.Execute(); err != nil {
						t.Fatalf("search failed: %v", err)
					}
					if searches.Load() != 1 {
						t.Fatalf("search requests=%d", searches.Load())
					}
					if tc.image && !tc.magnets && tc.format != "" && details.Load() != 1 {
						t.Fatalf("detail requests=%d", details.Load())
					}
					if tc.magnets && magnets.Load() != 1 {
						t.Fatalf("magnet requests=%d", magnets.Load())
					}
					if !strings.Contains(out.String(), "ABC-123") {
						t.Fatalf("missing search result: %s", out.String())
					}
				})
			}
		})
	}
}
