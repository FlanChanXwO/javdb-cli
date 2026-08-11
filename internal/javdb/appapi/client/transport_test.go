package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetJSONMapsEnvelopeErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantAuth   bool
		wantAction string
	}{
		{
			name:       "auth action",
			status:     http.StatusOK,
			body:       `{"success":false,"action":"Unauthorized","message":"expired"}`,
			wantAuth:   true,
			wantAction: "Unauthorized",
		},
		{
			name:       "ordinary action",
			status:     http.StatusOK,
			body:       `{"success":false,"action":"BadRequest","message":"invalid"}`,
			wantAction: "BadRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			c, err := New(Options{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			err = c.GetJSON("/api/test", nil, nil)
			if err == nil || !strings.Contains(err.Error(), tt.wantAction) {
				t.Fatalf("GetJSON error = %v, want action %q", err, tt.wantAction)
			}
			var authErr *AuthRequired
			if errors.As(err, &authErr) != tt.wantAuth {
				t.Fatalf("AuthRequired match = %t, want %t", errors.As(err, &authErr), tt.wantAuth)
			}
		})
	}
}

func TestGetJSONDecodesEnvelopeData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"value":"ok"}}`))
	}))
	defer server.Close()

	c, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := c.GetJSON("/api/test", nil, &got); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if got["value"] != "ok" {
		t.Fatalf("decoded data = %#v", got)
	}
}
