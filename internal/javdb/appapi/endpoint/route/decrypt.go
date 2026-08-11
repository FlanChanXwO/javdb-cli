// Package route 实现 JavDB App /startup 动态 API 域名的解密、候选提取与线路选择。
//
// 解密链路复刻 APK 1.9.28（common_tools.dart getDecryptString + backup_domains_data）：
//
//	key/iv = getDecryptString(input, 常量)   // MD5 还原 -> 逐字节相减 -> Base64
//	明文    = Base64 -> AES-CBC(key, iv) -> PKCS7 unpad -> UTF-8 JSON object
//
// 只读取 payload 的 apiDomains 字段并做规范化与去重；backupUrls（下载配置）与
// permanentWebDomain（网页域名）不作为 API 候选。
package route

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"
)

// 可诊断错误分类，便于表驱动测试与调用方区分失败阶段。
var (
	// ErrInvalidBase64 表示 backup_domains_data 或派生常量不是合法 Base64。
	ErrInvalidBase64 = errors.New("invalid base64")
	// ErrCipherLength 表示密文或 IV 长度不符合 AES-CBC 要求。
	ErrCipherLength = errors.New("cipher/iv length not multiple of block size")
	// ErrPadding 表示 PKCS7 padding 非法。
	ErrPadding = errors.New("invalid PKCS7 padding")
	// ErrInvalidUTF8 表示解密结果不是合法 UTF-8。
	ErrInvalidUTF8 = errors.New("decrypted payload is not valid UTF-8")
	// ErrInvalidJSON 表示解密结果不是 JSON object。
	ErrInvalidJSON = errors.New("decrypted payload is not a JSON object")
	// ErrFieldType 表示 apiDomains/backup_domains_data 字段类型非法。
	ErrFieldType = errors.New("domain field has invalid type")
	// ErrInvalidURL 表示 apiDomains 条目不合法 HTTP(S) URL。
	ErrInvalidURL = errors.New("domain entry is not a valid HTTP(S) URL")
)

// APK 1.9.28 中派生 backup_domains_data 的 AES key/IV 常量与其输入。
const (
	backupKeyInput = "30820"
	backupIVInput  = "astarte"

	backupKeyConst = "WzE5OSwxNjksMTYwLDE3NCwxOTksMTA2LDEyNCwxNzQsMTM4LDE3MywxNjIsMTQ5LDE5MCwx" +
		"NzksMTU3LDIwNiwxMjgsMjA5LDEyNSwxNzIsMTI4LDE4MiwxNjIsMTYxXQ=="
	backupIVConst = "WzE1MSwxNDMsMTI3LDEwMywxOTksMTQwLDIwMCwxNjksMTU3LDE2MiwxNjUsMTAxLDE5OCwx" +
		"NjMsMTc0LDE1NywyMDMsMTI1LDE1NiwxNjksMTQxLDIyMCwxMTEsMTYyXQ=="
)

// backupKey/backupIV 是一次性派生的 AES 材料；常量编译期固定，派生失败立即暴露。
var (
	backupKey, backupIV = deriveBackupKeyMaterial()
)

func deriveBackupKeyMaterial() (key, iv []byte) {
	k, err := getDecryptString(backupKeyInput, backupKeyConst)
	if err != nil {
		panic(fmt.Sprintf("route: derive backup key: %v", err))
	}
	i, err := getDecryptString(backupIVInput, backupIVConst)
	if err != nil {
		panic(fmt.Sprintf("route: derive backup iv: %v", err))
	}
	return []byte(k), []byte(i)
}

// getDecryptString 复刻 common_tools.dart 的字符串还原算法：
//
//	k     = md5(input).hexdigest()（32 个小写 hex 字符）
//	plain = json(base64decode(常量))（整数数组）
//	chars[i] = (plain[i] - ord(k[min(i, 31)])) & 0xff
//	result = base64decode(chars)
func getDecryptString(input, encoded string) (string, error) {
	sum := md5.Sum([]byte(input))
	keyHex := hex.EncodeToString(sum[:])

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode constant: %w", err)
	}
	var plain []int
	if err := json.Unmarshal(raw, &plain); err != nil {
		return "", fmt.Errorf("decode constant list: %w", err)
	}
	chars := make([]byte, len(plain))
	for i, p := range plain {
		idx := min(i, 31)
		chars[i] = byte(p - int(keyHex[idx]))
	}
	result, err := base64.StdEncoding.DecodeString(string(chars))
	if err != nil {
		return "", fmt.Errorf("decode derived constant: %w", err)
	}
	return string(result), nil
}

// DecryptBackupDomainsData 解密 startup 响应的 backup_domains_data 并返回 JSON payload。
// 链路：Base64 -> AES-CBC -> PKCS7 unpad -> UTF-8 JSON object。
func DecryptBackupDomainsData(encoded string) (map[string]any, error) {
	encrypted, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBase64, err)
	}
	if len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("%w: cipher length %d", ErrCipherLength, len(encrypted))
	}
	if len(backupIV) != aes.BlockSize {
		return nil, fmt.Errorf("%w: iv length %d", ErrCipherLength, len(backupIV))
	}
	block, err := aes.NewCipher(backupKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, backupIV).CryptBlocks(plain, encrypted)

	plain, err = pkcs7Unpad(plain)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPadding, err)
	}
	if !utf8.Valid(plain) {
		return nil, fmt.Errorf("%w", ErrInvalidUTF8)
	}
	var payload map[string]any
	if err := json.Unmarshal(plain, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	if payload == nil {
		return nil, fmt.Errorf("%w: decrypted value is null", ErrInvalidJSON)
	}
	return payload, nil
}

// APIHostsFromPayload 从解密后的 payload 提取规范化、去重后的 apiDomains 候选。
// 缺失 apiDomains 字段返回空切片（不报错）；字段类型或 URL 非法时返回可诊断错误。
func APIHostsFromPayload(payload map[string]any) ([]string, error) {
	raw, ok := payload["apiDomains"]
	if !ok {
		return nil, nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: apiDomains is %T", ErrFieldType, raw)
	}
	seen := make(map[string]bool, len(list))
	result := make([]string, 0, len(list))
	for _, item := range list {
		value, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%w: apiDomains entry is %T", ErrFieldType, item)
		}
		candidate, err := normalizeCandidate(value)
		if err != nil {
			return nil, err
		}
		if !seen[candidate] {
			seen[candidate] = true
			result = append(result, candidate)
		}
	}
	return result, nil
}

// APIHostsFromStartupData 从 /startup 的 data 提取动态 API 候选。
// backup_domains_data 缺失时返回空候选（不报错），便于调用方回退到已验证 bootstrap。
func APIHostsFromStartupData(startup map[string]any) ([]string, error) {
	raw, ok := startup["backup_domains_data"]
	if !ok {
		return nil, nil
	}
	encoded, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("%w: backup_domains_data is %T", ErrFieldType, raw)
	}
	payload, err := DecryptBackupDomainsData(encoded)
	if err != nil {
		return nil, err
	}
	return APIHostsFromPayload(payload)
}

// normalizeCandidate 校验并规范化单个 apiDomains 条目：去首尾空白与尾随 /，
// 必须是 http/https 且带 host，不允许 query/fragment。
func normalizeCandidate(value string) (string, error) {
	candidate := strings.TrimSpace(value)
	candidate = strings.TrimRight(candidate, "/")
	u, err := url.Parse(candidate)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidURL, value)
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return "", fmt.Errorf("%w: %q", ErrInvalidURL, value)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("%w: query/fragment not allowed: %q", ErrInvalidURL, value)
	}
	return candidate, nil
}

// pkcs7Unpad 校验并去除 AES 分组的 PKCS7 padding。
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("padded length %d", len(data))
	}
	padLen := int(data[len(data)-1])
	if padLen == 0 || padLen > aes.BlockSize || padLen > len(data) {
		return nil, fmt.Errorf("padding length %d", padLen)
	}
	for _, b := range data[len(data)-padLen:] {
		if int(b) != padLen {
			return nil, fmt.Errorf("padding bytes do not match length %d", padLen)
		}
	}
	return data[:len(data)-padLen], nil
}
