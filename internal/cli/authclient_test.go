package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
	"github.com/FlanChanXwO/javdb-cli/javdb"
)

// seedAuth writes a default account with token into the temp HOME.
func seedAuth(t *testing.T, token string) {
	t.Helper()
	fs, store, err := openAuth()
	if err != nil {
		t.Fatal(err)
	}
	store.Upsert(auth.Account{UserID: 1, Username: "u", Token: token}, true)
	if err := fs.Commit(store); err != nil {
		t.Fatal(err)
	}
}

func TestWithOptionalAuthClientNoTokenAnonymous(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	aio := &appIO{in: strings.NewReader(""), out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	calls := 0
	err := withOptionalAuthClient(&rootFlags{}, aio, func(c *javdb.Client) error {
		calls++
		if tok := c.Token(); tok != "" {
			t.Fatalf("anonymous call should have empty token, got %q", tok)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call with no saved token, got %d", calls)
	}
}

func TestWithOptionalAuthClientFallbackAnonymous(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedAuth(t, "saved-token")

	var errb bytes.Buffer
	aio := &appIO{in: strings.NewReader(""), out: &bytes.Buffer{}, err: &errb}

	calls := 0
	err := withOptionalAuthClient(&rootFlags{}, aio, func(c *javdb.Client) error {
		calls++
		switch calls {
		case 1:
			if tok := c.Token(); tok != "saved-token" {
				t.Fatalf("first call should carry saved token, got %q", tok)
			}
			return &javdb.AuthRequired{API: javdb.APIError{Action: "JWTVerificationError", Message: "bad"}}
		case 2:
			if tok := c.Token(); tok != "" {
				t.Fatalf("fallback call should be anonymous, got %q", tok)
			}
			return nil
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected fallback retry (2 calls), got %d", calls)
	}
	if !strings.Contains(errb.String(), "匿名") {
		t.Fatalf("expected fallback note on stderr, got %q", errb.String())
	}
}

func TestWithOptionalAuthClientNonAuthErrorNoFallback(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	seedAuth(t, "saved-token")

	aio := &appIO{in: strings.NewReader(""), out: &bytes.Buffer{}, err: &bytes.Buffer{}}

	calls := 0
	err := withOptionalAuthClient(&rootFlags{}, aio, func(c *javdb.Client) error {
		calls++
		return &javdb.APIError{Action: "NetworkError", Message: "boom"}
	})
	if err == nil {
		t.Fatal("expected non-auth error to be returned")
	}
	if calls != 1 {
		t.Fatalf("non-auth error must not retry, got %d calls", calls)
	}
}
