package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/image"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var testImage = append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, 0x00, 0x01, 0x02)

func testOptions() Options {
	return Options{
		HTTPClient:     http.DefaultClient,
		RequestTimeout: 60 * time.Second,
		Retries:        3,
		RetryWait:      10 * time.Millisecond,
	}
}

func okResponseJSON() []byte {
	return []byte(`{"results":[{"video_code":"SSIS-589","best_similarity":95.2,"frames":[
		{"image_name":"/frames/SSIS-589_01-04-53.jpg","similarity":95.2}]}]}`)
}

func TestBuiltinPostsRawMultipartAndNoCookies(t *testing.T) {
	var requestBody []byte
	var requestHeader http.Header
	var contentType string
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method, path = request.Method, request.URL.Path
		requestHeader = request.Header.Clone()
		contentType = request.Header.Get("Content-Type")
		body, _ := io.ReadAll(request.Body)
		requestBody = body
		_, _ = writer.Write(okResponseJSON())
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: 60 * time.Second,
		Retries:        3,
		RetryWait:      10 * time.Millisecond,
	})
	response, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.png"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if method != http.MethodPost {
		t.Errorf("request method = %s, want POST", method)
	}
	if path == "" {
		t.Error("request path is empty")
	}
	// 生产默认端点必须是官方 AVScan 地址（测试通过 Endpoint 覆盖到本地 server）。
	if BuiltinURL != "https://avscan.cc/search" {
		t.Errorf("BuiltinURL = %q", BuiltinURL)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Errorf("content type = %q", contentType)
	}
	if !bytes.Contains(requestBody, []byte(`name="file"`)) {
		t.Error("multipart body lacks the file field")
	}
	// 原始字节必须原样上传：不压缩、不转码、无 re-encode。
	if !bytes.Contains(requestBody, testImage) {
		t.Error("multipart body does not contain the original image bytes")
	}
	// 不得携带 Cookie 或浏览器指纹头。
	if cookie := requestHeader.Get("Cookie"); cookie != "" {
		t.Errorf("request sent Cookie header %q", cookie)
	}
	if len(response.Candidates) != 1 || response.Candidates[0].VideoCode != "SSIS-589" {
		t.Fatalf("unexpected candidates: %+v", response.Candidates)
	}
}

func TestBuiltinDerivesTimestampAndThumbnail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(okResponseJSON())
	}))
	defer server.Close()
	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        1,
	})
	response, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.png"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	frame := response.Candidates[0].Frames[0]
	if frame.Timestamp != "01:04:53" {
		t.Errorf("timestamp = %q, want 01:04:53", frame.Timestamp)
	}
	if frame.ThumbnailURL != "https://avscan.cc/thumb/SSIS-589/SSIS-589_01-04-53.webp" {
		t.Errorf("thumbnail = %q", frame.ThumbnailURL)
	}
	if frame.ImageName != "/frames/SSIS-589_01-04-53.jpg" {
		t.Errorf("image name = %q", frame.ImageName)
	}
}

func TestExternalUsesConfiguredHeadersAndUnifiedFieldsOnly(t *testing.T) {
	const secret = "Bearer super-secret-token-value"
	var sawHeader string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		sawHeader = request.Header.Get("Authorization")
		_, _ = writer.Write([]byte(`{"results":[{"video_code":"SSIS-589","best_similarity":88.0,"frames":[
			{"image_name":"custom/name.jpg","similarity":88.0,"timestamp":"00:12:34","thumbnail_url":"https://cdn.example/t.webp"}]}]}`))
	}))
	defer server.Close()

	external, err := New(Source{Name: "custom", URL: server.URL, Headers: map[string]string{"Authorization": secret}}, testOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	response, err := external.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if sawHeader != secret {
		t.Errorf("Authorization header = %q, want the configured value", sawHeader)
	}
	frame := response.Candidates[0].Frames[0]
	if frame.Timestamp != "00:12:34" || frame.ThumbnailURL != "https://cdn.example/t.webp" {
		t.Errorf("external frame fields not preserved: %+v", frame)
	}

	// 外部源省略 timestamp/thumbnail 时不得自行派生（不猜测命名规则）。
	server2 := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[{"video_code":"SSIS-589","frames":[{"image_name":"01_04_53.jpg"}]}]}`))
	}))
	defer server2.Close()
	external2, err := New(Source{Name: "custom", URL: server2.URL}, testOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	response2, err := external2.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	frame2 := response2.Candidates[0].Frames[0]
	if frame2.Timestamp != "" || frame2.ThumbnailURL != "" {
		t.Errorf("external source derived fields it must not: %+v", frame2)
	}
}

func TestRetriesOn429ThenSucceeds(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt < 3 {
			http.Error(writer, "rate limited", http.StatusTooManyRequests)
			return
		}
		_, _ = writer.Write(okResponseJSON())
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        3,
		RetryWait:      10 * time.Millisecond,
	})
	if _, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"}); err != nil {
		t.Fatalf("Search after 429 retries: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want exactly 3 total requests", got)
	}
}

func TestRetriesOnRequestTimeout(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		time.Sleep(200 * time.Millisecond)
		_, _ = writer.Write(okResponseJSON())
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: 30 * time.Millisecond,
		Retries:        3,
		RetryWait:      5 * time.Millisecond,
	})
	_, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"})
	if err == nil {
		t.Fatal("Search succeeded despite timeouts")
	}
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestNoRetryOnPermanentStatus(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		http.Error(writer, "backend down", http.StatusInternalServerError)
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        3,
		RetryWait:      10 * time.Millisecond,
	})
	if _, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"}); err == nil {
		t.Fatal("Search accepted HTTP 500")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (permanent status must not retry)", got)
	}
}

func TestNoRetryOnProtocolError(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		_, _ = writer.Write([]byte(`{"results": [not json`))
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        3,
		RetryWait:      10 * time.Millisecond,
	})
	if _, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"}); err == nil {
		t.Fatal("Search accepted malformed JSON")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (protocol errors must not retry)", got)
	}
}

func TestNoRetryOnCancellation(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		atomic.AddInt32(&attempts, 1)
		time.Sleep(200 * time.Millisecond)
		_, _ = writer.Write(okResponseJSON())
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        3,
		RetryWait:      10 * time.Millisecond,
	})
	if _, err := builtin.Search(ctx, Request{Image: testImage, Filename: "frame.jpg"}); err == nil {
		t.Fatal("Search accepted a cancelled context")
	}
	if got := atomic.LoadInt32(&attempts); got != 0 {
		t.Errorf("attempts = %d, want 0 (cancelled before any request)", got)
	}
}

func TestRejectsEmptyVideoCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[{"video_code":"","best_similarity":1.0,"frames":[]}]}`))
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        1,
	})
	_, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"})
	if err == nil {
		t.Fatal("Search accepted an empty video_code")
	}
	if !strings.Contains(err.Error(), "video_code") {
		t.Errorf("error should name the protocol field: %v", err)
	}
}

func TestRejectsUnsupportedExternalURL(t *testing.T) {
	if _, err := New(Source{Name: "custom", URL: "ftp://example.test/search"}, testOptions()); err == nil {
		t.Fatal("New accepted a non-HTTP source URL")
	}
	if _, err := New(Source{Name: "custom", URL: "not a url"}, testOptions()); err == nil {
		t.Fatal("New accepted a malformed source URL")
	}
}

func TestErrorsDoNotLeakHeaderValues(t *testing.T) {
	const secret = "Bearer top-secret-value-xyz"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "denied", http.StatusForbidden)
	}))
	defer server.Close()

	external, err := New(Source{Name: "custom", URL: server.URL, Headers: map[string]string{"Authorization": secret}}, testOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = external.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"})
	if err == nil {
		t.Fatal("Search accepted HTTP 403")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "top-secret") {
		t.Errorf("error leaks the header secret: %v", err)
	}
}

func TestResponsePreservesAllFramesInOrder(t *testing.T) {
	payload := map[string]any{
		"results": []map[string]any{
			{"video_code": "SSIS-589", "best_similarity": 95.2, "frames": []map[string]any{
				{"image_name": "a.jpg", "similarity": 95.2},
				{"image_name": "b.jpg", "similarity": 90.0, "timestamp": "00:01:02"},
				{"image_name": "c.jpg"},
			}},
		},
	}
	raw, _ := json.Marshal(payload)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(raw)
	}))
	defer server.Close()

	builtin := NewBuiltin(Options{
		HTTPClient: server.Client(),

		Endpoint:       server.URL,
		RequestTimeout: time.Second,
		Retries:        1,
	})
	response, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "frame.jpg"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	frames := response.Candidates[0].Frames
	if len(frames) != 3 {
		t.Fatalf("frame count = %d, want 3", len(frames))
	}
	if frames[0].ImageName != "a.jpg" || frames[2].ImageName != "c.jpg" {
		t.Errorf("frame order not preserved: %+v", frames)
	}
	if frames[1].Timestamp != "00:01:02" {
		t.Errorf("explicit timestamp lost: %q", frames[1].Timestamp)
	}
}

// TestResponseCarriesSourceName provider 响应必须填充实际 source 名称，
// 供 SDK/CLI 保留本次调用来源（meta.reverse_search.source 等）。
func TestResponseCarriesSourceName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(okResponseJSON())
	}))
	defer server.Close()
	builtin := NewBuiltin(Options{HTTPClient: server.Client(), Endpoint: server.URL, Retries: 1})
	response, err := builtin.Search(context.Background(), Request{Image: testImage, Filename: "f.jpg"})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if response.Source != BuiltinName {
		t.Errorf("builtin response source = %q, want %q", response.Source, BuiltinName)
	}

	external, err := New(Source{Name: "custom", URL: server.URL}, Options{HTTPClient: server.Client(), Retries: 1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	response, err = external.Search(context.Background(), Request{Image: testImage, Filename: "f.jpg"})
	if err != nil {
		t.Fatalf("Search external: %v", err)
	}
	if response.Source != "custom" {
		t.Errorf("external response source = %q, want custom", response.Source)
	}
}

// TestRejectsAbsoluteURLWithoutHost http:/... 这类非绝对 URL 在构造期拒绝。
func TestRejectsAbsoluteURLWithoutHost(t *testing.T) {
	for _, raw := range []string{"http:/search", "https:/", "http://"} {
		if _, err := New(Source{Name: "custom", URL: raw}, testOptions()); err == nil {
			t.Errorf("New accepted non-absolute URL %q", raw)
		}
	}
}

// TestMultipartPartContentTypeByMagic part 内联 Content-Type 按真实格式声明。
func TestMultipartPartContentTypeByMagic(t *testing.T) {
	body, _, err := buildMultipartBody([]byte{0xFF, 0xD8, 0xFF}, "f.jpg", image.JPEG)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("Content-Type: image/jpeg")) {
		t.Errorf("jpeg part header missing image/jpeg:\n%s", body)
	}
	body, _, err = buildMultipartBody([]byte{0x89, 'P', 'N', 'G'}, "f.png", image.PNG)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("Content-Type: image/png")) {
		t.Errorf("png part header missing image/png:\n%s", body)
	}
	body, _, err = buildMultipartBody([]byte("RIFF....WEBP"), "f.webp", image.WEBP)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("Content-Type: image/webp")) {
		t.Errorf("webp part header missing image/webp:\n%s", body)
	}
}
