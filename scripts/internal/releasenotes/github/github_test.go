package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/document"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

func TestClientReadsPullRequestAndFindsFirstMergedPullRequest(t *testing.T) {
	t.Parallel()

	mergedAt := time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/project/commits/abcdef012345/pulls":
			_ = json.NewEncoder(response).Encode([]model.GitHubPullRequest{{Number: 42}})
		case "/repos/owner/project/pulls/42":
			_ = json.NewEncoder(response).Encode(model.GitHubPullRequest{
				Number:   42,
				Title:    "feat: add APNG",
				Body:     "<!-- release-note\ncategory: Added\nbreaking: false\nsummary: Add APNG output.\nnone_reason:\n-->",
				HTMLURL:  "https://github.com/owner/project/pull/42",
				MergedAt: &mergedAt,
				User:     model.GitHubUser{Login: "new-contributor", Type: "User"},
			})
		case "/search/issues":
			if got, want := request.URL.Query().Get("q"), "repo:owner/project type:pr author:new-contributor is:merged"; got != want {
				http.Error(response, fmt.Sprintf("search query = %q, want %q", got, want), http.StatusInternalServerError)
				return
			}
			if got, want := request.URL.Query().Get("per_page"), "100"; got != want {
				http.Error(response, fmt.Sprintf("search per_page = %q, want %q", got, want), http.StatusInternalServerError)
				return
			}
			if got, want := request.URL.Query().Get("page"), "1"; got != want {
				http.Error(response, fmt.Sprintf("search page = %q, want %q", got, want), http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(response).Encode(model.GitHubPullRequestSearchResult{Items: []model.GitHubPullRequest{{Number: 42}}})
		default:
			http.Error(response, "unexpected GitHub API path "+request.URL.Path, http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	client := New(server.URL, "", server.Client())
	pulls, err := client.PullRequestsForCommit(context.Background(), "owner/project", "abcdef012345")
	if err != nil {
		t.Fatalf("pull requests for commit: %v", err)
	}
	if len(pulls) != 1 || pulls[0].Number != 42 {
		t.Fatalf("pulls = %#v", pulls)
	}
	pull, err := client.PullRequest(context.Background(), "owner/project", 42)
	if err != nil {
		t.Fatalf("pull request: %v", err)
	}
	if pull.User.Type != "User" || pull.User.Login == "owner" {
		t.Fatalf("pull %#v should be an eligible external contributor", pull)
	}
	first, err := client.FirstMergedPullRequest(context.Background(), "owner/project", pull.User.Login)
	if err != nil {
		t.Fatalf("first merged pull request: %v", err)
	}
	if first.Number != pull.Number {
		t.Fatalf("first merged pull = %#v, want #%d", first, pull.Number)
	}
	if _, err := document.ParseReleaseNoteDeclaration(pull.Body); err != nil {
		t.Fatalf("parse pull release note: %v", err)
	}
}

func TestValidRepository(t *testing.T) {
	if !ValidRepository("owner/project") {
		t.Fatal("owner/project must be valid")
	}
	if ValidRepository("project") || ValidRepository("owner/project/extra") {
		t.Fatal("malformed repository must be rejected")
	}
}
