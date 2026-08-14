package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/image"
)

// Classification 是 stdin 内容的输入类别。
type Classification int

const (
	// ClassificationImage 表示 JPEG/PNG/WEBP 图片字节。
	ClassificationImage Classification = iota
	// ClassificationNDJSON 表示 javdb.pipeline/v1 信封流。
	ClassificationNDJSON
	// ClassificationText 表示逐行纯文本 ref。
	ClassificationText
)

// Classify 按固定顺序识别 stdin 类别：图片 magic → NDJSON → 文本。
// 不消费输入：图片判定只 Peek 前 12 字节；NDJSON/文本判定读取全部剩余内容。
func Classify(reader *bufio.Reader) (Classification, []byte, error) {
	if reader == nil {
		return ClassificationText, nil, nil
	}
	header, err := reader.Peek(image.DetectHeaderSize())
	if err != nil && !errors.Is(err, io.EOF) {
		return ClassificationText, nil, fmt.Errorf("read stdin: %w", err)
	}
	if len(header) > 0 && image.DetectFormat(header) != image.Unknown {
		return ClassificationImage, nil, nil
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		return ClassificationText, nil, fmt.Errorf("read stdin: %w", err)
	}
	if looksLikeNDJSON(content) {
		return ClassificationNDJSON, content, nil
	}
	return ClassificationText, content, nil
}

// looksLikeNDJSON 判定首个非空行是否为合法 v1 信封。
func looksLikeNDJSON(content []byte) bool {
	for _, line := range bytes.Split(content, []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}
		var probe struct {
			Schema string `json:"schema"`
		}
		decoder := json.NewDecoder(strings.NewReader(trimmed))
		if err := decoder.Decode(&probe); err != nil {
			return false
		}
		return probe.Schema == Schema
	}
	return false
}
