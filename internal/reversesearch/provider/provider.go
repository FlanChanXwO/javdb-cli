// Package provider 实现反搜 provider：内置 AVScan adapter 与声明式外部
// HTTP adapter。两者都 POST 原始图片（multipart 字段固定为 file），解析统一
// 响应协议，并共享已确认的三次总请求与 30/60 秒退避。
//
// 与本地参考实现 scripts/avscan.py 的协议差异（已核对 2026-08-13 版本）：
//   - 不上传浏览器 UA 之外的自定义头；不携带 Cookie，不做 Cloudflare 绕过。
//   - 上传原始字节，不压缩/不转码/不重编码（参考实现会缩放到 1024 并转
//     JPEG）；multipart 内联 Content-Type 按真实 magic 声明。
//   - builtin 使用参考实现的命名规则从 image_name 派生 timestamp 与
//     thumbnail URL；外部 source 不派生，只接受统一响应字段。
//   - 不做 dHash 指纹库与黑帧过滤（本目标的缓存与隐私决策见 plan）。
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/image"
)

const (
	// BuiltinName 是内置 AVScan source 的保留名称。
	BuiltinName = "builtin"
	// BuiltinURL 是内置 AVScan 搜索端点。
	BuiltinURL = "https://avscan.cc/search"
)

// builtinUserAgent 与参考实现一致；avscan.cc 对普通 HTTP 请求直接开放，
// 该值只作为常规客户端标识，不构成任何绕过。
const builtinUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// Source 描述一个反搜源；Headers 必须已由调用方完成环境展开。
type Source struct {
	Name    string
	URL     string
	Headers map[string]string
}

// Frame 是一个候选结果中的一帧。
type Frame struct {
	ImageName    string
	Similarity   float64
	Timestamp    string
	ThumbnailURL string
}

// Candidate 是一个视频候选。
type Candidate struct {
	VideoCode  string
	Similarity float64
	Frames     []Frame
}

// Response 是 provider 规范化后的完整响应。
type Response struct {
	Source     string
	Candidates []Candidate
}

// Request 是一次反搜请求的原始图片输入。
type Request struct {
	Image    []byte
	Filename string
}

// Options 控制重试与超时；默认值与参考实现一致并可配置。
type Options struct {
	HTTPClient     *http.Client
	RequestTimeout time.Duration
	// Retries 是总请求次数（含首次），默认 3。
	Retries int
	// RetryWait 是首次退避时长；第二次等待为其两倍（30s/60s 语义）。
	RetryWait time.Duration
	// Endpoint 覆盖内置 AVScan 端点，仅供确定性测试使用；为空时使用
	// BuiltinURL。外部 source 使用自身的 URL。
	Endpoint string
}

// Provider 执行一次反搜。
type Provider interface {
	Search(context.Context, Request) (*Response, error)
}

// NewBuiltin 创建内置 AVScan provider。
func NewBuiltin(options Options) Provider {
	sourceURL := options.Endpoint
	if sourceURL == "" {
		sourceURL = BuiltinURL
	}
	return &httpProvider{
		source:        Source{Name: BuiltinName, URL: sourceURL},
		options:       withDefaults(options),
		deriveBuiltin: true,
	}
}

// New 创建声明式外部 HTTP provider；URL 必须是 HTTP(S)。
func New(source Source, options Options) (Provider, error) {
	if source.Name == "" {
		return nil, fmt.Errorf("reverse search source name is required")
	}
	parsed, err := url.Parse(source.URL)
	if err != nil || !isHTTPScheme(parsed.Scheme) || parsed.Host == "" {
		return nil, fmt.Errorf("reverse search source %q URL must be absolute HTTP(S), got %q", source.Name, source.URL)
	}
	return &httpProvider{
		source:  source,
		options: withDefaults(options),
	}, nil
}

type httpProvider struct {
	source        Source
	options       Options
	deriveBuiltin bool
}

func withDefaults(options Options) Options {
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	if options.RequestTimeout <= 0 {
		options.RequestTimeout = 60 * time.Second
	}
	if options.Retries <= 0 {
		options.Retries = 3
	}
	if options.RetryWait <= 0 {
		options.RetryWait = 30 * time.Second
	}
	return options
}

// Search 上传原始图片并解析统一响应。整个 provider 失败返回顶层 error，
// 调用方不得伪造空结果。
func (p *httpProvider) Search(ctx context.Context, request Request) (*Response, error) {
	body, contentType, err := buildMultipartBody(request.Image, p.partFilename(request.Filename), image.DetectFormat(request.Image))
	if err != nil {
		return nil, err
	}
	var last error
	for attempt := 1; attempt <= p.options.Retries; attempt++ {
		response, err := p.postOnce(ctx, body, contentType)
		if err == nil {
			return response, nil
		}
		last = err
		var retryable *retryableError
		if !errors.As(err, &retryable) || attempt == p.options.Retries {
			return nil, err
		}
		wait := p.options.RetryWait * time.Duration(attempt)
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, newError(StageCancel, "cancelled", "reverse search was cancelled while waiting to retry")
		}
	}
	return nil, last
}

func (p *httpProvider) partFilename(filename string) string {
	if p.deriveBuiltin {
		// 与参考实现一致：builtin 固定 filename=f.jpg。
		return "f.jpg"
	}
	if filename == "" {
		return "image"
	}
	return filename
}

func (p *httpProvider) postOnce(ctx context.Context, body []byte, contentType string) (*Response, error) {
	requestContext := ctx
	cancel := func() {}
	if p.options.RequestTimeout > 0 {
		requestContext, cancel = context.WithTimeout(ctx, p.options.RequestTimeout)
	}
	defer cancel()

	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, p.source.URL, bytes.NewReader(body))
	if err != nil {
		return nil, newError(StageRequest, "create", "create reverse search request: %v", err)
	}
	request.Header.Set("Content-Type", contentType)
	if p.deriveBuiltin {
		request.Header.Set("User-Agent", builtinUserAgent)
	}
	for name, value := range p.source.Headers {
		request.Header.Set(name, value)
	}
	response, err := p.options.HTTPClient.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, newError(StageCancel, "cancelled", "reverse search was cancelled")
		}
		if requestContext.Err() == context.DeadlineExceeded {
			return nil, &retryableError{cause: newError(StageTimeout, "timeout", "reverse search request timed out")}
		}
		if isRetryableNetworkError(err) {
			return nil, &retryableError{cause: newError(StageRequest, "transport", "reverse search request failed: %v", err)}
		}
		return nil, newError(StageRequest, "transport", "reverse search request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, &retryableError{cause: newError(StageStatus, "rate_limited", "reverse search source returned HTTP 429")}
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, newError(StageStatus, "unexpected", "reverse search source %q returned HTTP %s", p.source.Name, response.Status)
	}
	decoded, err := decodeResponse(response.Body)
	if err != nil {
		return nil, err
	}
	decoded.Source = p.source.Name
	if p.deriveBuiltin {
		deriveBuiltinFrames(decoded)
	}
	return decoded, nil
}

// 错误 stage；调用方据此区分请求、超时、状态、协议与取消错误。
const (
	StageRequest  = "request"
	StageTimeout  = "timeout"
	StageStatus   = "status"
	StageProtocol = "protocol"
	StageCancel   = "cancel"
)

// Error 是带稳定 stage/code 的 provider 错误。
type Error struct {
	Stage   string
	Code    string
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.Stage, e.Code, e.Message)
}

func newError(stage, code, format string, arguments ...any) *Error {
	return &Error{Stage: stage, Code: code, Message: fmt.Sprintf(format, arguments...)}
}

// retryableError 标记可以按退避策略重试的错误（429、单次超时、临时传输错误）。
type retryableError struct{ cause error }

func (e *retryableError) Unwrap() error { return e.cause }

func (e *retryableError) Error() string { return e.cause.Error() }

// isRetryableNetworkError 只覆盖临时传输错误；TLS 证书、协议等永久错误不重试。
func isRetryableNetworkError(err error) bool {
	var netErr interface{ Timeout() bool }
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	// 连接类瞬时失败（参考实现按 URLError/TimeoutError 退避的语义收敛）。
	for _, cause := range []error{
		syscall.ECONNREFUSED,
		syscall.ECONNRESET,
		syscall.EPIPE,
		syscall.EHOSTUNREACH,
		syscall.ENETUNREACH,
		syscall.ETIMEDOUT,
	} {
		if errors.Is(err, cause) {
			return true
		}
	}
	return false
}

func isHTTPScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

// buildMultipartBody 构建固定字段 file 的 multipart body，上传原始字节；
// part 内联 Content-Type 按真实 magic 声明（image/jpeg|png|webp）。
func buildMultipartBody(raw []byte, filename string, format image.Format) ([]byte, string, error) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	header.Set("Content-Type", partContentType(format))
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, "", newError(StageRequest, "create", "create multipart body: %v", err)
	}
	if _, err := part.Write(raw); err != nil {
		return nil, "", newError(StageRequest, "create", "write multipart body: %v", err)
	}
	if err := writer.Close(); err != nil {
		return nil, "", newError(StageRequest, "create", "close multipart body: %v", err)
	}
	return buffer.Bytes(), writer.FormDataContentType(), nil
}

// partContentType 把检测到的格式映射为 MIME 类型；未知格式返回 octet-stream。
func partContentType(format image.Format) string {
	switch format {
	case image.JPEG:
		return "image/jpeg"
	case image.PNG:
		return "image/png"
	case image.WEBP:
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// wireResponse 是统一响应协议（builtin 与外部 source 共用）。
type wireResponse struct {
	Results []wireCandidate `json:"results"`
}

type wireCandidate struct {
	VideoCode      string      `json:"video_code"`
	BestSimilarity float64     `json:"best_similarity"`
	Frames         []wireFrame `json:"frames"`
}

type wireFrame struct {
	ImageName    string  `json:"image_name"`
	Similarity   float64 `json:"similarity"`
	Timestamp    string  `json:"timestamp"`
	ThumbnailURL string  `json:"thumbnail_url"`
}

func decodeResponse(reader io.Reader) (*Response, error) {
	var wire wireResponse
	if err := json.NewDecoder(reader).Decode(&wire); err != nil {
		return nil, newError(StageProtocol, "decode", "decode reverse search response: %v", err)
	}
	response := &Response{Candidates: make([]Candidate, 0, len(wire.Results))}
	for _, candidate := range wire.Results {
		// 空或非法番号是协议错误：不吞错、不伪造空结果。
		if strings.TrimSpace(candidate.VideoCode) == "" {
			return nil, newError(StageProtocol, "video_code", "reverse search response contains an empty video_code")
		}
		normalized := Candidate{VideoCode: candidate.VideoCode, Similarity: candidate.BestSimilarity}
		for _, frame := range candidate.Frames {
			normalized.Frames = append(normalized.Frames, Frame{
				ImageName:    frame.ImageName,
				Similarity:   frame.Similarity,
				Timestamp:    frame.Timestamp,
				ThumbnailURL: frame.ThumbnailURL,
			})
		}
		response.Candidates = append(response.Candidates, normalized)
	}
	return response, nil
}

// deriveBuiltinFrames 按参考实现已知规则从 image_name 派生 timestamp 与
// thumbnail URL；外部 source 不调用本函数。
func deriveBuiltinFrames(response *Response) {
	for candidateIndex := range response.Candidates {
		candidate := &response.Candidates[candidateIndex]
		for frameIndex := range candidate.Frames {
			frame := &candidate.Frames[frameIndex]
			base := frameBaseName(frame.ImageName)
			if frame.Timestamp == "" {
				frame.Timestamp = deriveTimestamp(base)
			}
			if frame.ThumbnailURL == "" {
				frame.ThumbnailURL = "https://avscan.cc/thumb/" + candidate.VideoCode + "/" + base + ".webp"
			}
		}
	}
}

func frameBaseName(imageName string) string {
	last := imageName
	if index := strings.LastIndex(last, "/"); index >= 0 {
		last = last[index+1:]
	}
	if index := strings.LastIndex(last, "."); index >= 0 {
		last = last[:index]
	}
	return last
}

var timestampPattern = regexp.MustCompile(`^\d{2}:\d{2}(:\d{2})?$`)

func deriveTimestamp(base string) string {
	if !strings.Contains(base, "_") {
		return ""
	}
	suffix := base[strings.LastIndex(base, "_")+1:]
	candidate := strings.ReplaceAll(suffix, "-", ":")
	if !timestampPattern.MatchString(candidate) {
		return ""
	}
	return candidate
}
