package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Saved accounts must reach every command family; auth check must ask the server.
func TestAuthenticationBoundary(t *testing.T) {
	for _, command := range [][]string{{"actor", "1", "--json"}, {"browse", "--json"}, {"tags", "--refresh"}, {"lists", "search", "fixture", "--json"}} {
		t.Run(strings.Join(command, "-"), func(t *testing.T) {
			home := isolateHome(t)
			t.Setenv("LC_ALL", "C")
			dir := filepath.Join(home, ".javdb-cli")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"default_user_id":1,"accounts":[{"user_id":1,"token":"fixture-token"}]}`), 0600); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer fixture-token" {
					fmt.Fprint(w, `{"success":0,"action":"JWTVerificationError","message":"Invalid Signature"}`)
					return
				}
				fmt.Fprint(w, `{"success":1,"data":{"actor":{"id":"1"},"movies":[],"lists":[],"categories":[]}}`)
			}))
			defer server.Close()
			var out, errOut bytes.Buffer
			args := append([]string{"--host", server.URL}, command...)
			if code := Run(args, strings.NewReader(""), &out, &errOut); code != 0 {
				t.Fatalf("exit=%d: %s", code, errOut.String())
			}
		})
	}
	t.Run("server-rejected-JWT", func(t *testing.T) {
		home := isolateHome(t)
		t.Setenv("LC_ALL", "C")
		dir := filepath.Join(home, ".javdb-cli")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		jwt := "fixture." + base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":1,"username":"fixture"}`)) + ".fixture"
		if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(fmt.Sprintf(`{"default_user_id":1,"accounts":[{"user_id":1,"token":%q}]}`, jwt)), 0600); err != nil {
			t.Fatal(err)
		}
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			fmt.Fprint(w, `{"success":0,"action":"JWTVerificationError","message":"Invalid Signature"}`)
		}))
		defer server.Close()
		var out, errOut bytes.Buffer
		if code := Run([]string{"--host", server.URL, "auth", "check", "--json"}, strings.NewReader(""), &out, &errOut); code != 1 || calls.Load() != 1 || !strings.Contains(out.String(), `"ok":false`) {
			t.Fatalf("exit=%d calls=%d stdout=%s stderr=%s", code, calls.Load(), out.String(), errOut.String())
		}
	})
}

func TestAnonymousGuidanceAndOrderedBatchDiagnostics(t *testing.T) {
	for _, input := range []string{"fixture", "fixture\nother\n"} {
		t.Run(fmt.Sprintf("%d-bytes", len(input)), func(t *testing.T) {
			isolateHome(t)
			t.Setenv("LC_ALL", "C")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"success":0,"action":"JWTVerificationError","message":"Invalid Signature"}`)
			}))
			defer server.Close()
			var out, errOut bytes.Buffer
			args := []string{"--host", server.URL, "--log-format", "json", "search"}
			reader := strings.NewReader("")
			count := 1
			if strings.Contains(input, "\n") {
				reader = strings.NewReader(input)
				count = 2
			} else {
				args = append(args, input)
			}
			if code := Run(args, reader, &out, &errOut); code != 1 || out.Len() != 0 {
				t.Fatalf("exit=%d stdout=%s", code, out.String())
			}
			lines := strings.Split(strings.TrimSpace(errOut.String()), "\n")
			expected := count
			if count > 1 {
				expected++
			}
			if len(lines) != expected {
				t.Fatalf("duplicate/missing diagnostics: %s", errOut.String())
			}
			for i, line := range lines {
				var record struct {
					Level, Msg, Error, Command string
					InputIndex                 int `json:"input_index"`
				}
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record.Level != "ERROR" || record.Command != "javdb search" {
					t.Fatalf("wrong command/level: %s", line)
				}
				if i < count && record.InputIndex != i+1 {
					t.Fatalf("wrong input position: %s", line)
				}
				if i < count && (!strings.Contains(record.Msg, "javdb auth login") || !strings.Contains(record.Error, "JWTVerificationError")) {
					t.Fatalf("missing user guidance or original cause: %s", line)
				}
			}
		})
	}
}
