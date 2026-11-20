package session

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/session/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// Concurrent replays share one login and persist its token before successful results escape.
func TestConcurrentAuthenticationRecovery(t *testing.T) {
	isolateHomeForAuth(t)
	t.Setenv("JAVDB_AUTO_RELOGIN", "true")
	seedAuth(t, auth.Account{UserID: 1, Username: "fixture", Password: "fixture-password", Token: "expired"})
	const workers = 4
	var oldRequests, logins atomic.Int32
	gate := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/sessions":
			logins.Add(1)
			fmt.Fprint(w, `{"success":1,"data":{"token":"fresh"}}`)
		case "/api/v2/search":
			if r.Header.Get("Authorization") != "Bearer fresh" {
				if oldRequests.Add(1) == workers {
					close(gate)
				}
				<-gate
				fmt.Fprint(w, `{"success":0,"action":"JWTVerificationError","message":"Invalid Signature"}`)
				return
			}
			fmt.Fprint(w, `{"success":1,"data":{"movies":[]}}`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c, err := NewWithDefaultToken(&Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			if _, err := c.Search(context.Background(), "fixture", javdb.SearchOptions{}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if logins.Load() != 1 {
		t.Fatalf("logins=%d, want one", logins.Load())
	}
	_, store, err := authstore.Open()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.Default()
	if err != nil || saved.Token != "fresh" {
		t.Fatalf("token was not persisted: %v", err)
	}
}

// Login may renew server-side session state without changing the opaque token string.
func TestAuthenticationRecoveryDoesNotRequireTokenRotation(t *testing.T) {
	isolateHomeForAuth(t)
	t.Setenv("JAVDB_AUTO_RELOGIN", "true")
	seedAuth(t, auth.Account{UserID: 1, Username: "fixture", Password: "fixture-password", Token: "same-token"})
	var signedIn atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/sessions" {
			signedIn.Store(true)
			fmt.Fprint(w, `{"success":1,"data":{"token":"same-token"}}`)
			return
		}
		if !signedIn.Load() {
			fmt.Fprint(w, `{"success":0,"action":"JWTVerificationError","message":"Invalid Signature"}`)
			return
		}
		fmt.Fprint(w, `{"success":1,"data":{"movies":[]}}`)
	}))
	defer server.Close()
	c, err := NewWithDefaultToken(&Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Search(context.Background(), "fixture", javdb.SearchOptions{}); err != nil {
		t.Fatal(err)
	}
}
