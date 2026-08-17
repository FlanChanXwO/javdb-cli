package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/github"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

func TestNewContributorExcludesOwnerAndBots(t *testing.T) {
	t.Parallel()

	for _, pull := range []model.GitHubPullRequest{
		{User: model.GitHubUser{Login: "owner", Type: "User"}},
		{User: model.GitHubUser{Login: "dependabot[bot]", Type: "Bot"}},
	} {
		if IsExternalContributor(pull, "owner") {
			t.Fatalf("pull %#v must not be listed as a new external contributor", pull)
		}
	}
	if !IsExternalContributor(model.GitHubPullRequest{User: model.GitHubUser{Login: "other", Type: "User"}}, "owner") {
		t.Fatal("an external user should be eligible before historical PR lookup")
	}
}

func TestPRValidateReadsEventPayload(t *testing.T) {
	t.Parallel()

	eventPath := filepath.Join(t.TempDir(), "event.json")
	body, err := json.Marshal(map[string]any{"pull_request": map[string]any{"body": "<!-- release-note\ncategory: Documentation\nbreaking: false\nsummary: Clarify the release workflow.\nnone_reason:\n-->"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(eventPath, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePullRequestEvent(eventPath); err != nil {
		t.Fatalf("validate PR event: %v", err)
	}
}

func TestPullRequestsForCommitResolvesSquashMergeByExactPRCommit(t *testing.T) {
	t.Parallel()

	const commit = "a7f6e39e2e744fff4ca386f1aae6c7e99b640bc2"
	mergedAt := time.Date(2026, time.August, 17, 0, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/project/commits/" + commit + "/pulls":
			http.Error(response, "association unavailable", http.StatusInternalServerError)
		case "/repos/owner/project/pulls/29":
			_ = json.NewEncoder(response).Encode(model.GitHubPullRequest{
				Number:         29,
				Title:          "docs(release): prepare v0.7.2",
				Body:           "<!-- release-note\ncategory: Documentation\nbreaking: false\nsummary: Prepare release notes.\nnone_reason:\n-->",
				HTMLURL:        "https://github.com/owner/project/pull/29",
				MergedAt:       &mergedAt,
				MergeCommitSHA: commit,
				User:           model.GitHubUser{Login: "owner", Type: "User"},
			})
		default:
			http.Error(response, "unexpected GitHub API path "+request.URL.Path, http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	pulls, err := pullRequestsForCommit(
		context.Background(),
		github.New(server.URL, "", server.Client()),
		"owner/project",
		commit,
		"docs(release): prepare v0.7.2 (#29)",
	)
	if err != nil {
		t.Fatalf("resolve squash PR: %v", err)
	}
	if len(pulls) != 1 || pulls[0].Number != 29 {
		t.Fatalf("pulls = %#v, want PR #29", pulls)
	}
}

func TestPullRequestsForCommitPreservesUnattributedLookupFailure(t *testing.T) {
	t.Parallel()

	const commit = "abcdef0123456789"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "association unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := pullRequestsForCommit(
		context.Background(),
		github.New(server.URL, "", server.Client()),
		"owner/project",
		commit,
		"direct maintenance commit",
	)
	if err == nil || !strings.Contains(err.Error(), "lookup PRs for commit "+commit) {
		t.Fatalf("error = %v, want lookup failure", err)
	}
}
