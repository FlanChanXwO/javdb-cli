package route

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// backupDomainsData 是 APK 1.9.28 的真实 /startup 样本，用于验证 key/IV 派生与字段边界。
const backupDomainsData = "JCxJQTR1DerICeuy4lmmWJuj2sRqgbDdvL2Nru5I6BmGb+GmAKKAUbjeLL1r+rFe" +
	"Oxq+Kb3g2MOSXYpvd9dA7Pds+G6brFTtRy7EQ0s4DkIaUfAzoKgMWldPRI/0IvUj" +
	"OvVkn1t0/nUIEz2LTWmcKx5sj3BVtIV5XEiRtS8fUGvVSddw6Fy7g9nJ/iN5OxFC" +
	"ypbRPK0dd6+09Vx3ALU/9kI39VeBlNZE7/Vjnr2nc0MZg3PIZHCt9dlldO9uS7GM" +
	"LU+LHXFq29VbyGGkXxlOuO+dE4ejYK1CJ9Qx14FuR1xWx3p8rOHo1INDE7LmqgZy" +
	"/3vDlRY8hHbdDr81tKWBAS/PXcOakVZGNuEiOf6OKtQR9J3M44MUStw+k5AZ9jh0" +
	"KhblvYeTdA79l1b+byubUqyDLP5XiEkyT2yQ8JTB/wHfH6Otg5/5NoI22nODaQjK" +
	"UaFDDnzr0S2Vwbp0uu68GAov458mHuuIUleBSI4TGqA="

func TestDerivedKeyMaterialLengths(t *testing.T) {
	if len(backupKey) != aes.BlockSize {
		t.Fatalf("backupKey length = %d, want %d", len(backupKey), aes.BlockSize)
	}
	if len(backupIV) != aes.BlockSize {
		t.Fatalf("backupIV length = %d, want %d", len(backupIV), aes.BlockSize)
	}
}

// TestDecryptRealFixtureExtractsOnlyAPIDomains 用真实加密样本验证：只提取两个 apidd.*，
// 排除 backupUrls 与 permanentWebDomain。
func TestDecryptRealFixtureExtractsOnlyAPIDomains(t *testing.T) {
	payload, err := DecryptBackupDomainsData(backupDomainsData)
	if err != nil {
		t.Fatalf("DecryptBackupDomainsData() error = %v", err)
	}
	domains, err := APIHostsFromPayload(payload)
	if err != nil {
		t.Fatalf("APIHostsFromPayload() error = %v", err)
	}
	want := []string{"https://apidd.spthgb.com", "https://apidd.czssdgz.com"}
	if strings.Join(domains, ",") != strings.Join(want, ",") {
		t.Fatalf("domains = %v, want %v", domains, want)
	}
	for _, excluded := range []string{"https://javdb.com", "cos", "backupUrls"} {
		for _, domain := range domains {
			if strings.Contains(domain, excluded) {
				t.Fatalf("domain %q should not be present (excluded %q)", domain, excluded)
			}
		}
	}
}

func TestAPIHostsFromStartupDataMissingBackupIsEmpty(t *testing.T) {
	domains, err := APIHostsFromStartupData(map[string]any{"settings": "x"})
	if err != nil {
		t.Fatalf("APIHostsFromStartupData() error = %v", err)
	}
	if len(domains) != 0 {
		t.Fatalf("domains = %v, want empty", domains)
	}
}

func TestAPIHostsFromStartupDataRoundTrip(t *testing.T) {
	domains, err := APIHostsFromStartupData(map[string]any{"backup_domains_data": backupDomainsData})
	if err != nil {
		t.Fatalf("APIHostsFromStartupData() error = %v", err)
	}
	if len(domains) != 2 || domains[0] != "https://apidd.spthgb.com" {
		t.Fatalf("domains = %v", domains)
	}
}

func TestAPIHostsDeduplicatesPreservingOrder(t *testing.T) {
	payload := map[string]any{
		"apiDomains": []any{
			"https://a.example.com/",  // 规范化去尾 /
			"https://a.example.com",   // 去重
			" https://b.example.com ", // 去首尾空白
			"https://b.example.com/",
		},
	}
	domains, err := APIHostsFromPayload(payload)
	if err != nil {
		t.Fatalf("APIHostsFromPayload() error = %v", err)
	}
	want := []string{"https://a.example.com", "https://b.example.com"}
	if strings.Join(domains, ",") != strings.Join(want, ",") {
		t.Fatalf("domains = %v, want %v", domains, want)
	}
}

func TestDecryptErrorBoundaries(t *testing.T) {
	t.Run("invalid base64", func(t *testing.T) {
		_, err := DecryptBackupDomainsData("!!!not-base64!!!")
		if !errors.Is(err, ErrInvalidBase64) {
			t.Fatalf("error = %v, want ErrInvalidBase64", err)
		}
	})
	t.Run("cipher length not multiple", func(t *testing.T) {
		enc := base64.StdEncoding.EncodeToString([]byte{1, 2, 3})
		_, err := DecryptBackupDomainsData(enc)
		if !errors.Is(err, ErrCipherLength) {
			t.Fatalf("error = %v, want ErrCipherLength", err)
		}
	})
	t.Run("invalid padding", func(t *testing.T) {
		_, err := DecryptBackupDomainsData(encryptedWithBrokenPadding())
		if !errors.Is(err, ErrPadding) {
			t.Fatalf("error = %v, want ErrPadding", err)
		}
	})
	t.Run("invalid utf8", func(t *testing.T) {
		_, err := DecryptBackupDomainsData(encryptForTest(bytes.Repeat([]byte{0xFF}, 15), 1))
		if !errors.Is(err, ErrInvalidUTF8) {
			t.Fatalf("error = %v, want ErrInvalidUTF8", err)
		}
	})
	t.Run("not json object", func(t *testing.T) {
		_, err := DecryptBackupDomainsData(encryptForTest([]byte("hello world"), 5))
		if !errors.Is(err, ErrInvalidJSON) {
			t.Fatalf("error = %v, want ErrInvalidJSON", err)
		}
	})
	t.Run("json null payload", func(t *testing.T) {
		_, err := DecryptBackupDomainsData(encryptForTest([]byte("null"), 12))
		if !errors.Is(err, ErrInvalidJSON) {
			t.Fatalf("error = %v, want ErrInvalidJSON", err)
		}
	})
}

func TestAPIHostsFieldTypeErrors(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]any
		want    error
	}{
		{"apiDomains not a list", map[string]any{"apiDomains": "nope"}, ErrFieldType},
		{"apiDomains entry not string", map[string]any{"apiDomains": []any{123}}, ErrFieldType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := APIHostsFromPayload(tc.payload)
			if !errors.Is(err, tc.want) {
				t.Fatalf("error = %v, want %v", err, tc.want)
			}
		})
	}
	_, err := APIHostsFromStartupData(map[string]any{"backup_domains_data": 42})
	if !errors.Is(err, ErrFieldType) {
		t.Fatalf("startup field type error = %v, want ErrFieldType", err)
	}
}

func TestAPIHostsURLValidation(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"unsupported scheme", "ftp://x.example.com"},
		{"missing host", "https://"},
		{"relative", "/no/scheme"},
		{"query not allowed", "https://x.example.com?a=1"},
		{"fragment not allowed", "https://x.example.com#frag"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := APIHostsFromPayload(map[string]any{"apiDomains": []any{tc.url}})
			if !errors.Is(err, ErrInvalidURL) {
				t.Fatalf("error = %v, want ErrInvalidURL for %q", err, tc.url)
			}
		})
	}
}

// encryptForTest 用真实派生 key/IV 对 plaintext 做 PKCS7 填充后 AES-CBC 加密并 Base64。
func encryptForTest(plaintext []byte, padLen int) string {
	padded := append([]byte{}, plaintext...)
	padded = append(padded, bytes.Repeat([]byte{byte(padLen)}, padLen)...)
	block, err := aes.NewCipher(backupKey)
	if err != nil {
		panic(err)
	}
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, backupIV).CryptBlocks(out, padded)
	return base64.StdEncoding.EncodeToString(out)
}

// encryptedWithBrokenPadding 返回一段密文，解密后最后字节被翻转为 0x00（非法 padLen）。
func encryptedWithBrokenPadding() string {
	encrypted := bytes.Repeat([]byte{0xAA}, 15)
	encrypted = append(encrypted, 0x01) // 合法 pad=1
	block, err := aes.NewCipher(backupKey)
	if err != nil {
		panic(err)
	}
	out := make([]byte, len(encrypted))
	cipher.NewCBCEncrypter(block, backupIV).CryptBlocks(out, encrypted)
	out[len(out)-1] ^= 0x01 // 翻转最后字节 -> 解出 padLen=0
	return base64.StdEncoding.EncodeToString(out)
}
