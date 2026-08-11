// Package codec 负责 App API 的 JSON、JWT、用户 ID 和响应数组解析。
package codec

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// ObjectArray 解码 JSON object 数组，保持 App API 原有的 nil/非法输入语义。
func ObjectArray(raw json.RawMessage) []map[string]any {
	return jsonx.ObjectArray(raw)
}

// ObjectSlice 将 App API 响应中的通用 slice 转成 object slice。
func ObjectSlice(v any) []map[string]any {
	return jsonx.ObjectSlice(v)
}

// RawString 读取 JSON object 中的字符串字段；非字符串值按原始 JSON 文本回退。
func RawString(m map[string]json.RawMessage, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	return strings.Trim(string(v), `"`)
}

// UserIDFromMap 从 JSON object 的常见 id 字段解析非零用户 ID 和名称。
func UserIDFromMap(m map[string]json.RawMessage) (int64, string, bool) {
	for _, key := range []string{"id", "user_id", "uid"} {
		if raw, ok := m[key]; ok {
			if id, ok := ParseID(raw); ok {
				name := RawString(m, "username")
				if name == "" {
					name = RawString(m, "email")
				}
				if name == "" {
					name = RawString(m, "name")
				}
				return id, name, true
			}
		}
	}
	return 0, "", false
}

// ParseID 从 JSON raw value 解析非零用户 ID；保留 API 对浮点数的截断行为。
func ParseID(raw json.RawMessage) (int64, bool) {
	var n int64
	if json.Unmarshal(raw, &n) == nil && n != 0 {
		return n, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n != 0 {
			return n, true
		}
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil && f != 0 {
		return int64(f), true
	}
	return 0, false
}

// UserIDFromJWT 从 JWT payload 解码常见用户 ID 和名称，不校验签名。
func UserIDFromJWT(token string) (int64, string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0, "", false
	}
	payload, err := DecodeSegment(parts[1])
	if err != nil {
		return 0, "", false
	}
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil {
		return 0, "", false
	}
	if id, name, ok := userIDFromClaims(claims, []string{"user_id", "uid", "id", "sub"}); ok {
		return id, name, true
	}
	if nested, ok := claims["user"].(map[string]any); ok {
		return userIDFromClaims(nested, []string{"id", "user_id", "uid"})
	}
	return 0, "", false
}

func userIDFromClaims(claims map[string]any, keys []string) (int64, string, bool) {
	for _, key := range keys {
		if v, ok := claims[key]; ok {
			switch t := v.(type) {
			case float64:
				if t != 0 {
					return int64(t), ClaimString(claims, "username", "email", "name"), true
				}
			case string:
				if n, err := strconv.ParseInt(t, 10, 64); err == nil && n != 0 {
					return n, ClaimString(claims, "username", "email", "name"), true
				}
			}
		}
	}
	return 0, "", false
}

// ClaimString 返回 claims 中第一个非空字符串字段。
func ClaimString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

// DecodeSegment 解码不带 padding 的 base64url JWT segment。
func DecodeSegment(segment string) ([]byte, error) {
	s := segment
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	return decodeStd(s)
}

func decodeStd(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}
