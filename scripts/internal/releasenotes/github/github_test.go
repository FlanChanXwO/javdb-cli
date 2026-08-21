package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

func TestClientReadsPullRequestSources(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/project/commits/abcdef012345/pulls":
			_ = json.NewEncoder(response).Encode([]model.GitHubPullRequest{{Number: 42}})
		case "/repos/owner/project/pulls/42":
			_ = json.NewEncoder(response).Encode(model.GitHubPullRequest{
				Number:  42,
				Title:   "feat: add APNG",
				HTMLURL: "https://github.com/owner/project/pull/42",
				User:    model.GitHubUser{Login: "maintainer", Type: "User"},
			})
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
	if pull.Title != "feat: add APNG" || pull.HTMLURL != "https://github.com/owner/project/pull/42" {
		t.Fatalf("pull %#v should preserve source metadata", pull)
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
