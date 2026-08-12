package release

import (
	"strings"
	"testing"
)

// TestNewReleaseHTTPClientNeverLeaksCredentials 验证 url.Parse 失败时错误消息不回显
// 可能携带 userinfo 凭据的原始 proxy。
func TestNewReleaseHTTPClientNeverLeaksCredentials(t *testing.T) {
	if _, err := NewReleaseHTTPClient("http://review-user:review-secret@host:badport"); err == nil {
		t.Fatal("NewReleaseHTTPClient() accepted malformed proxy")
	} else if strings.Contains(err.Error(), "review-secret") {
		t.Fatalf("NewReleaseHTTPClient() leaks credentials: %v", err)
	}
}
