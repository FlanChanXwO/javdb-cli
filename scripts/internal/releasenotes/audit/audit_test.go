package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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
