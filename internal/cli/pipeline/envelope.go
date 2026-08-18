// Package pipeline 实现 javdb.pipeline/v1 机器协议核心：typed envelope、
// 严格 NDJSON/文本解码、输入分类与输出模式。
//
// 命令之间通过 NDJSON 组合：生产者按固定 schema 输出逐条 envelope，消费者按
// kind 选择 id 或 ref。默认输出文本；显式 --ndjson / --json 互斥。
package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Schema 是管道信封的固定 schema。
const Schema = "javdb.pipeline/v1"

// Kind 是稳定的记录类别；消费者只接受声明匹配的 kind。
type Kind string

const (
	KindMovie     Kind = "movie"
	KindActor     Kind = "actor"
	KindSeries    Kind = "series"
	KindMaker     Kind = "maker"
	KindDirector  Kind = "director"
	KindCode      Kind = "code"
	KindList      Kind = "list"
	KindAccount   Kind = "account"
	KindComment   Kind = "comment"
	KindMagnet    Kind = "magnet"
	KindDownload  Kind = "download"
	KindConfigKey Kind = "config_key"
	KindError     Kind = "error"
)

// Envelope 是管道记录。
type Envelope struct {
	Schema string         `json:"schema"`
	Kind   Kind           `json:"kind"`
	Ref    string         `json:"ref,omitempty"`
	ID     string         `json:"id,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// New 构造一个 schema 校验通过的信封。
func New(kind Kind, ref, id string) Envelope {
	return Envelope{Schema: Schema, Kind: kind, Ref: ref, ID: id}
}

// WithData 返回附带 data 的信封副本。
func (e Envelope) WithData(data map[string]any) Envelope {
	e.Data = data
	return e
}

// WithMeta 返回附带 meta 的信封副本。
func (e Envelope) WithMeta(meta map[string]any) Envelope {
	e.Meta = meta
	return e
}

// ErrorEnvelope 构造原位错误信封；不含 secret。
func ErrorEnvelope(input Envelope, command, stage, code, message string) Envelope {
	return Envelope{
		Schema: Schema,
		Kind:   KindError,
		Ref:    input.Ref,
		ID:     input.ID,
		Data: map[string]any{
			"input_ref": input.Ref,
			"input_id":  input.ID,
			"command":   command,
			"stage":     stage,
			"code":      code,
			"message":   message,
		},
	}
}

// ErrorMessage 提取上游错误信封的可展示 message；缺失、null 或非字符串值
// 统一使用稳定回退，避免把协议数据的类型错误泄露成 fmt 的诊断占位符。
func ErrorMessage(envelope Envelope) string {
	if message, ok := envelope.Data["message"].(string); ok && message != "" {
		return message
	}
	return "upstream pipeline error"
}

// Validate 校验信封 schema 与 kind；ref/id 至少一个存在。
func (e Envelope) Validate() error {
	if e.Schema != Schema {
		return fmt.Errorf("envelope schema %q is not %q", e.Schema, Schema)
	}
	if !IsKind(e.Kind) {
		return fmt.Errorf("envelope has unsupported kind %q", e.Kind)
	}
	if e.Ref == "" && e.ID == "" {
		return fmt.Errorf("envelope must carry a ref or an id")
	}
	return nil
}

// IsKind 报告 kind 是否为稳定集合之一。
func IsKind(kind Kind) bool {
	switch kind {
	case KindMovie, KindActor, KindSeries, KindMaker, KindDirector,
		KindCode, KindList, KindAccount, KindComment, KindMagnet,
		KindDownload, KindConfigKey, KindTag, KindError:
		return true
	default:
		return false
	}
}

// DecodeNDJSON 严格解码一条 NDJSON 记录；拒绝非对象行、尾随数据与非法信封。
func DecodeNDJSON(line string) (Envelope, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Envelope{}, errEmptyLine()
	}
	decoder := json.NewDecoder(strings.NewReader(trimmed))
	var envelope Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return Envelope{}, fmt.Errorf("decode envelope: %w", err)
	}
	if decoder.More() {
		return Envelope{}, fmt.Errorf("envelope line contains trailing data")
	}
	if err := envelope.Validate(); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}

// DecodeText 把一行纯文本解码为 ref（kind 由消费者命令决定）。
func DecodeText(line string) (Envelope, error) {
	ref := strings.TrimSpace(line)
	if ref == "" {
		return Envelope{}, errEmptyLine()
	}
	return New("", ref, ""), nil
}

// ParseBatch 把 stdin 内容按分类解码为信封序列：
//   - 输入分类固定顺序为图片 magic、NDJSON、文本（由 Classify 完成）；
//   - NDJSON 模式逐行严格解码，混合/非法行返回带行号的错误；
//   - 文本模式逐行转 ref。
func ParseBatch(content []byte, kind Kind) ([]Envelope, error) {
	lines := strings.Split(string(content), "\n")
	envelopes := make([]Envelope, 0, len(lines))
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var envelope Envelope
		var err error
		switch kind {
		case KindNDJSONInput:
			envelope, err = DecodeNDJSON(line)
		default:
			envelope, err = DecodeText(line)
			if err == nil {
				envelope.Kind = kind
			}
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", index+1, err)
		}
		envelopes = append(envelopes, envelope)
	}
	return envelopes, nil
}

// KindNDJSONInput 是 ParseBatch 的哨兵：表示逐行 NDJSON 严格解码。
const KindNDJSONInput Kind = "ndjson"

func errEmptyLine() error {
	return errors.New("empty line")
}
