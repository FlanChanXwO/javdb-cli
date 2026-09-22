package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseTrustSourceUsesImmutableTagOnDefaultBranch(t *testing.T) {
	// The hook runner exports GIT_DIR/GIT_WORK_TREE/GIT_INDEX_FILE while a hook
	// executes, and git gives those precedence over `-C`. Production git helpers
	// therefore pin the repository root themselves; this test passes the
	// fixture path explicitly and asserts no operation escapes it.
	t.Setenv("PATH", "/usr/bin:/bin"+string(os.PathListSeparator)+os.Getenv("PATH"))
	repo := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", repo}, args...)...)
		command.Env = append(os.Environ(),
			"GIT_DIR="+filepath.Join(repo, ".git"),
			"GIT_WORK_TREE="+repo,
			"GIT_INDEX_FILE="+filepath.Join(repo, ".git", "index"),
		)
		body, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, body)
		}
		return strings.TrimSpace(string(body))
	}
	git("init", "-b", "main")
	git("config", "user.name", "release-test")
	git("config", "user.email", "release-test@example.invalid")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "README.md")
	git("commit", "-m", "fixture")
	git("tag", "v1.2.3")
	commit := git("rev-parse", "HEAD")
	git("update-ref", "refs/remotes/origin/main", commit)

	trust, err := verifySourceTrust(repo, "v1.2.3", "main")
	if err != nil {
		t.Fatalf("verify source trust: %v", err)
	}
	if trust.Commit != commit || trust.Version != "1.2.3" {
		t.Fatalf("source trust = %#v, want commit %s version 1.2.3", trust, commit)
	}

	git("checkout", "--orphan", "other")
	git("rm", "-rf", ".")
	if err := os.WriteFile(filepath.Join(repo, "other.txt"), []byte("other\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "other.txt")
	git("commit", "-m", "other")
	git("tag", "v1.2.4")
	if _, err := verifySourceTrust(repo, "v1.2.4", "main"); err == nil {
		t.Fatal("source trust accepted tag outside the default-branch ancestry")
	}
}

// TestReleaseTrustGitHelpersIgnoreInheritedGitEnv is the regression test for the
// repo-damaging bug: an inherited GIT_DIR overrides `-C`, so the release git
// helpers must pin their own repository root. Without the fix this test writes
// the v1.2.3 tag and the release-test identity into whatever repository the
// ambient GIT_DIR points at.
func TestReleaseTrustGitHelpersIgnoreInheritedGitEnv(t *testing.T) {
	fixture := t.TempDir()
	if err := runGit(fixture, "init", "-b", "main"); err != nil {
		t.Fatalf("init fixture: %v", err)
	}
	if err := runGit(fixture, "config", "user.name", "fixture"); err != nil {
		t.Fatalf("configure name: %v", err)
	}
	if err := runGit(fixture, "config", "user.email", "fixture@example.invalid"); err != nil {
		t.Fatalf("configure email: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "README.md"), []byte("fixture\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runGit(fixture, "add", "README.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := runGit(fixture, "commit", "-m", "fixture"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Simulate the hook runner: point the ambient git environment at an
	// unrelated decoy repository that must stay untouched.
	decoy := t.TempDir()
	if err := runGit(decoy, "init", "-b", "main"); err != nil {
		t.Fatalf("init decoy: %v", err)
	}
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
	t.Setenv("GIT_WORK_TREE", decoy)
	if err := runGit(fixture, "tag", "v1.2.3"); err != nil {
		t.Fatalf("tag fixture: %v", err)
	}

	toplevel, err := captureGit(fixture, "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("capture toplevel: %v", err)
	}
	if !samePath(toplevel, fixture) {
		t.Fatalf("git helpers resolved %q, want the fixture root %q", toplevel, fixture)
	}
	if _, err := captureGit(decoy, "rev-parse", "--verify", "refs/tags/v1.2.3"); err == nil {
		t.Fatal("fixture tag leaked into the ambient GIT_DIR repository")
	}
}

// samePath compares two paths that git and Go may render differently: git
// reports forward slashes even on Windows, Go uses the platform separator, and
// the temporary directory can be reached through a symlink. Normalize the
// separators explicitly rather than via filepath.ToSlash so the helper behaves
// the same on every platform, then resolve symlinks and compare folded.
func samePath(left, right string) bool {
	canonical := func(value string) string {
		if resolved, err := filepath.EvalSymlinks(value); err == nil {
			value = resolved
		}
		// Normalize to forward slashes on every platform: on Unix the Windows
		// fixture path contains literal backslashes that filepath.Clean leaves
		// alone, while on Windows git returns forward slashes.
		value = strings.ReplaceAll(value, `\`, "/")
		for strings.Contains(value, "//") {
			value = strings.ReplaceAll(value, "//", "/")
		}
		return strings.TrimSuffix(strings.ToLower(value), "/")
	}
	return canonical(left) == canonical(right)
}

func TestReleaseTrustPublishedReleaseAndHandoff(t *testing.T) {
	t.Parallel()
	commit := strings.Repeat("a", 40)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(writer, "missing auth", http.StatusUnauthorized)
			return
		}
		switch request.URL.Path {
		case "/repos/FlanChanXwO/javdb-cli/releases/tags/v1.2.3":
			fmt.Fprint(writer, `{"tag_name":"v1.2.3","draft":false,"published_at":"2026-09-20T00:00:00Z"}`)
		case "/repos/FlanChanXwO/javdb-cli/actions/runs/42":
			fmt.Fprintf(writer, `{"name":"Release","conclusion":"success","head_sha":%q}`, commit)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	client := githubReleaseClient{BaseURL: server.URL, Token: "test-token", HTTP: server.Client()}
	if err := verifyPublishedRelease(client, "FlanChanXwO/javdb-cli", "v1.2.3"); err != nil {
		t.Fatalf("verify published release: %v", err)
	}
	if err := verifyReleaseHandoff(client, "FlanChanXwO/javdb-cli", 42, "Release", commit); err != nil {
		t.Fatalf("verify release handoff: %v", err)
	}
	if err := verifyReleaseHandoff(client, "FlanChanXwO/javdb-cli", 42, "Release", ""); err != nil {
		t.Fatalf("verify recovery handoff without head binding: %v", err)
	}
}
