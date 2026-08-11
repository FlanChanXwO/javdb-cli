package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

// TestGetJSONContextCancellation 验证 context 取消会让阻塞中的请求立刻返回，并且取消后
// 不再发起重试（默认重试计数下也不会空等退避）。
func TestGetJSONContextCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))
	defer server.Close()
	defer close(release) // 无论断言结果如何都让 handler 退出，避免测试内 goroutine 悬挂

	c, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- c.GetJSONContext(ctx, "/api/v1/startup", nil, nil)
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("GetJSONContext error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetJSONContext did not return after cancellation")
	}
}

// TestGetJSONContextCancelTearsDownServerRequest 验证取消后服务端请求 context 也被拆除，
// 证明没有悬挂的服务端请求 goroutine 持有连接。
func TestGetJSONContextCancelTearsDownServerRequest(t *testing.T) {
	started := make(chan struct{})
	serverObserved := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done() // 客户端断开连接后触发
		close(serverObserved)
	}))
	defer server.Close()

	c, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- c.GetJSONContext(ctx, "/api/v1/startup", nil, nil)
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("GetJSONContext error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("client did not return after cancellation")
	}
	select {
	case <-serverObserved:
	case <-time.After(5 * time.Second):
		t.Fatal("server never observed the client disconnect")
	}
}

// TestGetJSONContextZeroRetriesSingleRequest 验证显式零重试只发起一次真实请求（测速样本
// 不被重试污染），且错误原样返回。
func TestGetJSONContextZeroRetriesSingleRequest(t *testing.T) {
	zero := 0
	var count int32
	server := newClosingServer(&count)
	defer server.Close()

	c, err := New(Options{Host: server.URL, Retries: &zero})
	if err != nil {
		t.Fatal(err)
	}
	err = c.GetJSONContext(context.Background(), "/api/v1/startup", nil, nil)
	if err == nil {
		t.Fatal("expected network error from closed connection")
	}
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

// TestGetJSONDefaultRetriesUnchanged 验证 nil Retries 保持默认重试：网络错误下共 1+2 次尝试。
func TestGetJSONDefaultRetriesUnchanged(t *testing.T) {
	var count int32
	server := newClosingServer(&count)
	defer server.Close()

	c, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = c.GetJSONContext(context.Background(), "/api/v1/startup", nil, nil)
	if err == nil {
		t.Fatal("expected network error from closed connection")
	}
	if got := atomic.LoadInt32(&count); got != 1+defaultRetries {
		t.Fatalf("requests = %d, want %d", got, 1+defaultRetries)
	}
}

// TestGetJSONContextPropagatesEnvelopeError 验证 context-aware 路径同样映射 App envelope 错误。
func TestGetJSONContextPropagatesEnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":false,"action":"BadRequest","message":"nope"}`))
	}))
	defer server.Close()

	c, err := New(Options{Host: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	err = c.GetJSONContext(context.Background(), "/api/v1/startup", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "BadRequest") {
		t.Fatalf("GetJSONContext error = %v, want action BadRequest", err)
	}
}

// newClosingServer 返回一个立即关闭底层连接（不发 HTTP 响应）的服务器，让客户端侧产生
// 可重试的网络错误，并记录每个请求的命中次数。
func newClosingServer(count *int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(count, 1)
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close()
	}))
}
