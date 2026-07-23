package update

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubReleaseClientSelectsHighestCompatibleReleaseAcrossPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/releases":
			writer.Header().Set("Link", fmt.Sprintf("<%s/page-2>; rel=\"next\"", "http://"+request.Host))
			_, _ = writer.Write([]byte(`[
                {"tag_name":"v0.2.0-beta.1","prerelease":true},
                {"tag_name":"v0.1.1","prerelease":false},
                {"tag_name":"not-a-release","prerelease":false}
            ]`))
		case "/page-2":
			_, _ = writer.Write([]byte(`[{"tag_name":"v0.2.0","prerelease":false}]`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client, err := NewGitHubReleaseClient(ReleaseClientOptions{HTTPClient: server.Client(), Endpoint: server.URL + "/releases"})
	if err != nil {
		t.Fatal(err)
	}
	stable, err := client.Check(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if stable == nil || stable.TagName != "v0.2.0" {
		t.Fatalf("stable release = %#v", stable)
	}
	prerelease, err := client.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if prerelease == nil || prerelease.TagName != "v0.2.0" {
		t.Fatalf("prerelease-enabled release = %#v", prerelease)
	}
}

func TestGitHubReleaseClientExcludesPrereleaseWhenRequested(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`[
            {"tag_name":"v0.3.0-beta.1","prerelease":true},
            {"tag_name":"v0.2.1-beta.1","prerelease":false},
            {"tag_name":"v0.2.0","prerelease":false}
        ]`))
	}))
	defer server.Close()
	client, err := NewGitHubReleaseClient(ReleaseClientOptions{HTTPClient: server.Client(), Endpoint: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	stable, err := client.Check(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if stable == nil || stable.TagName != "v0.2.0" {
		t.Fatalf("stable release = %#v", stable)
	}
	withPrerelease, err := client.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if withPrerelease == nil || withPrerelease.TagName != "v0.3.0-beta.1" {
		t.Fatalf("prerelease release = %#v", withPrerelease)
	}
}
