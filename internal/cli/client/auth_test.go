package client

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

func isolateHomeForAuth(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(dir))
	t.Setenv("HOMEPATH", strings.TrimPrefix(dir, filepath.VolumeName(dir)))
	return dir
}

func seedAuth(t *testing.T, account auth.Account) {
	t.Helper()
	fs, store, err := authstore.Open()
	if err != nil {
		t.Fatal(err)
	}
	store.Upsert(account, true)
	if err := fs.Commit(store); err != nil {
		t.Fatal(err)
	}
}

func TestWithOptionalAuthNoTokenAnonymous(t *testing.T) {
	isolateHomeForAuth(t)
	var errb bytes.Buffer

	calls := 0
	err := WithOptionalAuth(&invocation.RootOptions{}, &errb, func(c *javdb.Client) error {
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

func TestWithOptionalAuthFallbackAnonymous(t *testing.T) {
	isolateHomeForAuth(t)
	seedAuth(t, auth.Account{UserID: 1, Username: "u", Token: "saved-token"})

	var errb bytes.Buffer
	calls := 0
	err := WithOptionalAuth(&invocation.RootOptions{}, &errb, func(c *javdb.Client) error {
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

func TestWithOptionalAuthNonAuthErrorNoFallback(t *testing.T) {
	isolateHomeForAuth(t)
	seedAuth(t, auth.Account{UserID: 1, Username: "u", Token: "saved-token"})

	var errb bytes.Buffer
	calls := 0
	err := WithOptionalAuth(&invocation.RootOptions{}, &errb, func(c *javdb.Client) error {
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

func TestWithRequiredAuthNoDefaultAccount(t *testing.T) {
	isolateHomeForAuth(t)
	var errb bytes.Buffer
	err := WithRequiredAuth(&invocation.RootOptions{}, &errb, func(*javdb.Client) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "no default account; run: javdb auth login") {
		t.Fatalf("error = %v, want no-default-account", err)
	}
}

func TestWithRequiredAuthEmptyToken(t *testing.T) {
	isolateHomeForAuth(t)
	seedAuth(t, auth.Account{UserID: 1, Username: "u", Token: ""})
	var errb bytes.Buffer
	err := WithRequiredAuth(&invocation.RootOptions{}, &errb, func(*javdb.Client) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "default account has no token; run: javdb auth login") {
		t.Fatalf("error = %v, want empty-token", err)
	}
}

func TestWithRequiredAuthAuthRequiredWithoutAutoRelogin(t *testing.T) {
	isolateHomeForAuth(t)
	seedAuth(t, auth.Account{UserID: 1, Username: "u", Token: "expired"})
	var errb bytes.Buffer
	calls := 0
	err := WithRequiredAuth(&invocation.RootOptions{}, &errb, func(*javdb.Client) error {
		calls++
		return &javdb.AuthRequired{API: javdb.APIError{Action: "JWTVerificationError", Message: "bad"}}
	})
	if err == nil || !strings.Contains(err.Error(), "token expired or invalid; run: javdb auth login (or: javdb config set auto_relogin true)") {
		t.Fatalf("error = %v, want auto-relogin disabled message", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestWithRequiredAuthNoSavedPassword(t *testing.T) {
	isolateHomeForAuth(t)
	t.Setenv("JAVDB_AUTO_RELOGIN", "true")
	seedAuth(t, auth.Account{UserID: 1, Username: "u", Token: "expired"})
	var errb bytes.Buffer
	err := WithRequiredAuth(&invocation.RootOptions{}, &errb, func(*javdb.Client) error {
		return &javdb.AuthRequired{API: javdb.APIError{Action: "JWTVerificationError", Message: "bad"}}
	})
	if err == nil || !strings.Contains(err.Error(), "token expired and no saved password; run: javdb auth login") {
		t.Fatalf("error = %v, want no-saved-password", err)
	}
}

func TestWithRequiredAuthAutoReloginPersistsToken(t *testing.T) {
	isolateHomeForAuth(t)
	t.Setenv("JAVDB_AUTO_RELOGIN", "true")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/sessions" {
			http.Error(w, "unexpected request", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"token":"new-token"}}`))
	}))
	defer server.Close()

	seedAuth(t, auth.Account{UserID: 1, Username: "u", Password: "pw", Token: "expired"})

	var errb bytes.Buffer
	calls := 0
	err := WithRequiredAuth(&invocation.RootOptions{Host: server.URL}, &errb, func(c *javdb.Client) error {
		calls++
		switch calls {
		case 1:
			if tok := c.Token(); tok != "expired" {
				t.Fatalf("first call token = %q, want expired", tok)
			}
			return &javdb.AuthRequired{API: javdb.APIError{Action: "JWTVerificationError", Message: "bad"}}
		case 2:
			if tok := c.Token(); tok != "new-token" {
				t.Fatalf("relogin call token = %q, want new-token", tok)
			}
			return nil
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithRequiredAuth error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected re-login retry (2 calls), got %d", calls)
	}
	if !strings.Contains(errb.String(), "重新登录") {
		t.Fatalf("expected relogin note on stderr, got %q", errb.String())
	}
	// token 持久化
	_, store, err := authstore.Open()
	if err != nil {
		t.Fatal(err)
	}
	acc, err := store.Default()
	if err != nil || acc.Token != "new-token" {
		t.Fatalf("persisted account = %+v, %v; want token new-token", acc, err)
	}
}
