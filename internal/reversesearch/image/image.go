// Package image 提供反搜图片的统一原始读取入口：本地路径、HTTP(S) URL 与
// stdin 流。只按 magic bytes 接受 JPEG/PNG/WEBP，保留原始字节，并在流式读取
// 中强制 8 MiB 上限；文件名与响应 Content-Type 只用于诊断，不作为信任依据。
package image

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// MaxSize 是图片字节上限：恰好 8 MiB 合法，读到第 8 MiB 后的第一个字节
// 立即返回明确错误。
const MaxSize = 8 << 20

// maxRedirects 是图片 URL 重定向的硬上限。
const maxRedirects = 10

// Format 是支持的图片格式。
type Format int

const (
	// Unknown 表示无法识别为 JPEG/PNG/WEBP。
	Unknown Format = iota
	// JPEG 对应 FF D8 FF magic。
	JPEG
	// PNG 对应 89 50 4E 47 0D 0A 1A 0A magic。
	PNG
	// WEBP 对应 RIFF....WEBP magic。
	WEBP
)

func (f Format) String() string {
	switch f {
	case JPEG:
		return "jpeg"
	case PNG:
		return "png"
	case WEBP:
		return "webp"
	default:
		return "unknown"
	}
}

// Image 是已验证的原始图片字节与派生元数据。
type Image struct {
	Bytes    []byte
	Format   Format
	Filename string
	SHA256   [32]byte
}

// 错误 stage，用于区分输入、下载、状态、格式、大小与取消错误；调用方不得
// 把失败回退成文本搜索。
const (
	StageInput    = "input"
	StageDownload = "download"
	StageStatus   = "status"
	StageFormat   = "format"
	StageSize     = "size"
	StageCancel   = "cancel"
)

// Error 是带稳定 stage/code 的图片错误。
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

var (
	pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	riffPrefix   = []byte("RIFF")
	webpMarker   = []byte("WEBP")
)

// DetectFormat 按开头字节判定图片格式；至少需要对应格式的完整 magic。
func DetectFormat(header []byte) Format {
	if len(header) >= 3 && header[0] == 0xFF && header[1] == 0xD8 && header[2] == 0xFF {
		return JPEG
	}
	if len(header) >= len(pngSignature) && bytes.Equal(header[:len(pngSignature)], pngSignature) {
		return PNG
	}
	if len(header) >= 12 && bytes.Equal(header[:4], riffPrefix) && bytes.Equal(header[8:12], webpMarker) {
		return WEBP
	}
	return Unknown
}

// ReadFile 读取本地普通文件并校验；路径必须是常规文件。
func ReadFile(path string) (*Image, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, newError(StageInput, "open", "open image %q: %v", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, newError(StageInput, "type", "image source %q is not a regular file", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, newError(StageInput, "open", "open image %q: %v", path, err)
	}
	defer file.Close()
	return readStreamWithStage(file, path, StageInput)
}

// ReadStream 从任意 reader（stdin、内存等）流式读取并校验。
func ReadStream(reader io.Reader, sourceName string) (*Image, error) {
	return readStreamWithStage(reader, sourceName, StageInput)
}

func readStreamWithStage(reader io.Reader, sourceName, readErrorStage string) (*Image, error) {
	if reader == nil {
		return nil, newError(StageInput, "read", "image source %q is nil", sourceName)
	}
	var buffer bytes.Buffer
	limited := io.LimitReader(reader, MaxSize+1)
	read, err := buffer.ReadFrom(limited)
	if err != nil {
		return nil, newError(readErrorStage, "read", "read image %q: %v", sourceName, err)
	}
	if read > MaxSize {
		return nil, newError(StageSize, "too_large", "image %q exceeds the 8 MiB limit", sourceName)
	}
	format := DetectFormat(buffer.Bytes())
	if format == Unknown {
		return nil, newError(StageFormat, "unsupported", "image %q is not JPEG, PNG or WEBP", sourceName)
	}
	return &Image{
		Bytes:    buffer.Bytes(),
		Format:   format,
		Filename: sourceName,
		SHA256:   sha256.Sum256(buffer.Bytes()),
	}, nil
}

// ReadURL 下载并校验 HTTP(S) 图片；允许私网地址（SDK 文档提示服务端嵌入者
// 自行施加网络边界）。初始 URL 与每次重定向都重新校验 scheme，最终响应必须
// 是 2xx，body 流式受 8 MiB 限制。client 复用调用方注入的 transport（含全局
// 代理配置）。
func ReadURL(ctx context.Context, client *http.Client, rawURL string) (*Image, error) {
	if client == nil {
		client = http.DefaultClient
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || !isHTTPScheme(parsed.Scheme) {
		return nil, newError(StageInput, "scheme", "image URL %q must be HTTP(S)", rawURL)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, newError(StageInput, "request", "create image request for %q: %v", rawURL, err)
	}
	response, err := redirectCheckingClient(client).Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, newError(StageCancel, "cancelled", "image download was cancelled")
		}
		return nil, newError(StageDownload, "request", "download image %q: %v", rawURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return nil, newError(StageStatus, "unexpected", "image URL %q returned HTTP %s", rawURL, response.Status)
	}
	return readStreamWithStage(response.Body, rawURL, StageDownload)
}

func isHTTPScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

// redirectCheckingClient 复用原 transport（保留代理配置），逐跳复检 scheme
// 并限制重定向次数。
func redirectCheckingClient(base *http.Client) *http.Client {
	transport := base.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &http.Client{
		Transport: transport,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			if !isHTTPScheme(request.URL.Scheme) {
				return errors.New("redirect to a non-HTTP(S) scheme")
			}
			return nil
		},
	}
}
