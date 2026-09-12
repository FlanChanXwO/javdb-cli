package download_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli"
)

func TestAssetsAndDownloadAliasExecuteSameLocalAssetFlow(t *testing.T) {
	isolateDownloadAliasHome(t)
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x01}
	var detailIDs []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case strings.HasPrefix(request.URL.Path, "/api/v4/movies/"):
			id := strings.TrimPrefix(request.URL.Path, "/api/v4/movies/")
			detailIDs = append(detailIDs, id)
			_ = json.NewEncoder(writer).Encode(map[string]any{"success": true, "data": map[string]any{
				"movie": map[string]any{"id": id, "thumb_url": server.URL + "/media/thumbnail.jpg"},
			}})
		case request.URL.Path == "/media/thumbnail.jpg":
			_, _ = writer.Write(thumbnail)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	for _, commandName := range []string{"assets", "download"} {
		target := filepath.Join(t.TempDir(), commandName+".jpg")
		var out, errb bytes.Buffer
		root := cli.New(strings.NewReader(""), &out, &errb)
		root.SetArgs([]string{"--host", server.URL, commandName, "movie-id", "--id", "--thumbnail", target})
		if err := root.Execute(); err != nil {
			t.Fatalf("%s execute: %v; stderr=%q", commandName, err, errb.String())
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("%s output: %v", commandName, err)
		}
		if !bytes.Equal(got, thumbnail) {
			t.Fatalf("%s output = %x, want %x", commandName, got, thumbnail)
		}
		if !strings.Contains(out.String(), "thumbnail\t"+target) {
			t.Fatalf("%s output summary = %q", commandName, out.String())
		}
	}
	if len(detailIDs) != 2 || detailIDs[0] != "movie-id" || detailIDs[1] != "movie-id" {
		t.Fatalf("detail requested for ids %v, want [movie-id movie-id]", detailIDs)
	}
}

func isolateDownloadAliasHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
}
