package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/github"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

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
