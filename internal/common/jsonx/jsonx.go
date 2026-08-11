// Package jsonx 提供纯 JSON 编解码 helper。
//
// 该包只做字节级转换，不接收 io.Writer、不写输出、不含 CLI 文案、不吞掉编码错误；
// CLI/SDK/App API 的 JSON 字节契约统一由此保证。
package jsonx

import (
	"bytes"
	"encoding/json"
)

// ObjectArray 解码 JSON object 数组；空值、null、非法 JSON 或非数组返回 nil。
// 数组中的 null 元素按 encoding/json 的既有语义保留为 nil map。
func ObjectArray(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// ObjectSlice 将常见的 map slice 或可 JSON 编码的 slice 转换为 object slice。
// []any 中的非 map 元素沿用 App API 原有实现，直接忽略。
func ObjectSlice(v any) []map[string]any {
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, x := range t {
			if m, ok := x.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil
		}
		return out
	}
}

// RawString 读取 JSON 原始标量字符串字段：若原始值以引号包裹则剥除外层引号，
// 否则返回原始文本。不擅自 unescape 或规范化内容。
func RawString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	value := string(raw)
	if len(value) >= 2 && value[0] == '"' {
		return value[1 : len(value)-1]
	}
	return value
}

// MarshalLine 以紧凑 JSON 编码 value，SetEscapeHTML(false)，
// 成功时返回恰好包含一个尾随换行的字节。编码失败原样返回错误。
func MarshalLine(value any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
