package history

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/github"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

func writeReleaseNote(t *testing.T, root, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestSyncHistoryDryRunAndApplyPreservesAssets(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeReleaseNote(t, directory, "en.md", "# v1.1.0 — 2026-07-30\n\n## Added\n\n- Added one. ([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))\n\n**Full Changelog**: [v1.0.0...v1.1.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.0.0...v1.1.0)\n")
	writeReleaseNote(t, directory, "zh-CN.md", "# v1.1.0 — 2026-07-30\n\n## 新增\n\n- 新增一。([#42](https://github.com/FlanChanXwO/javdb-cli/pull/42))\n\n**完整变更**：[v1.0.0...v1.1.0](https://github.com/FlanChanXwO/javdb-cli/compare/v1.0.0...v1.1.0)\n")
	var releaseBody string
	var patchCount int
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/repos/owner/project/releases/tags/v1.1.0":
			_ = json.NewEncoder(response).Encode(model.GitHubRelease{ID: 99, TagName: "v1.1.0", Body: releaseBody, Assets: []model.GitHubReleaseAsset{{Name: "javdb-cli.tar.gz"}}})
		case request.Method == http.MethodPatch && request.URL.Path == "/repos/owner/project/releases/99":
			patchCount++
			var payload model.GitHubReleaseWrite
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				http.Error(response, "decode patch: "+err.Error(), http.StatusBadRequest)
				return
			}
			releaseBody = payload.Body
			response.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(response).Encode(model.GitHubRelease{ID: 99, TagName: "v1.1.0", Body: releaseBody, Assets: []model.GitHubReleaseAsset{{Name: "javdb-cli.tar.gz"}}})
		default:
			http.Error(response, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	config := Config{Repository: "owner/project", Version: "1.1.0", Directory: directory, Client: github.New(server.URL, "", server.Client())}
	if err := SyncHistoricalRelease(context.Background(), config); err != nil {
		t.Fatalf("dry-run sync: %v", err)
	}
	if patchCount != 0 {
		t.Fatalf("dry-run patched %d times", patchCount)
	}
	config.Apply = true
	if err := SyncHistoricalRelease(context.Background(), config); err != nil {
		t.Fatalf("apply sync: %v", err)
	}
	if patchCount != 1 || !strings.Contains(releaseBody, "# English") || !strings.Contains(releaseBody, "# 简体中文") {
		t.Fatalf("sync result patchCount=%d body=%q", patchCount, releaseBody)
	}
}

func TestSyncHistoryCreatesMissingHistoricalReleaseWithoutAssets(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	writeReleaseNote(t, directory, "en.md", "# v0.4.0 — 2026-07-18\n\n## Maintenance\n\n- Historical maintenance. ([`abcdef0`](https://github.com/FlanChanXwO/javdb-cli/commit/abcdef0123456789))\n\n**Full Changelog**: [v0.3.0...v0.4.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.3.0...v0.4.0)\n")
	writeReleaseNote(t, directory, "zh-CN.md", "# v0.4.0 — 2026-07-18\n\n## 维护\n\n- 历史维护。([`abcdef0`](https://github.com/FlanChanXwO/javdb-cli/commit/abcdef0123456789))\n\n**完整变更**：[v0.3.0...v0.4.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.3.0...v0.4.0)\n")
	var created model.GitHubReleaseWrite
	var createdRelease model.GitHubRelease
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/repos/owner/project/releases/tags/v0.4.0" && createdRelease.ID == 0:
			response.WriteHeader(http.StatusNotFound)
			_, _ = response.Write([]byte(`{"message":"Not Found"}`))
		case request.Method == http.MethodPost && request.URL.Path == "/repos/owner/project/releases":
			if err := json.NewDecoder(request.Body).Decode(&created); err != nil {
				http.Error(response, "decode create: "+err.Error(), http.StatusBadRequest)
				return
			}
			createdRelease = model.GitHubRelease{ID: 40, TagName: created.TagName, Name: created.Name, Body: created.Body}
			_ = json.NewEncoder(response).Encode(createdRelease)
		case request.Method == http.MethodGet && request.URL.Path == "/repos/owner/project/releases/tags/v0.4.0":
			_ = json.NewEncoder(response).Encode(createdRelease)
		default:
			http.Error(response, "unexpected request", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	err := SyncHistoricalRelease(context.Background(), Config{
		Repository: "owner/project", Version: "0.4.0", Directory: directory, Apply: true,
		Client: github.New(server.URL, "", server.Client()),
	})
	if err != nil {
		t.Fatalf("create historical release: %v", err)
	}
	if created.TagName != "v0.4.0" || created.Name != "v0.4.0" || created.Draft || strings.Contains(created.Body, "assets") {
		t.Fatalf("created release payload = %#v", created)
	}
}
