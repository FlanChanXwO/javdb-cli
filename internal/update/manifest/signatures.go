package manifest

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
)

// Signature 是单个公钥对发布清单字节的 Ed25519 签名。
// 字段顺序即规范 JSON 编码顺序，禁止重排。
type Signature struct {
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"`
}

// Signatures 是发布清单的签名文件。
// 字段顺序即规范 JSON 编码顺序，禁止重排。
type Signatures struct {
	Schema     string      `json:"schema"`
	Signatures []Signature `json:"signatures"`
}

// KeyID 派生公钥标识：原始 32-byte Ed25519 公钥的 SHA-256 小写十六进制。
func KeyID(publicKey ed25519.PublicKey) string {
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:])
}

// Sign 使用 seed 派生的私钥对 message 签名。
func Sign(seed []byte, message []byte) (Signature, error) {
	if len(seed) != ed25519.SeedSize {
		return Signature{}, fmt.Errorf("Ed25519 seed must be %d bytes", ed25519.SeedSize)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	signature := Signature{
		KeyID:     KeyID(privateKey.Public().(ed25519.PublicKey)),
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, message)),
	}
	return signature, nil
}

// Verify 校验 message 上的单个签名；任何编码或长度问题都显式报错。
func Verify(message []byte, signature Signature, publicKey ed25519.PublicKey) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("Ed25519 public key must be %d bytes", ed25519.PublicKeySize)
	}
	raw, err := base64.StdEncoding.DecodeString(signature.Signature)
	if err != nil {
		return fmt.Errorf("decode signature for key %q: %w", signature.KeyID, err)
	}
	if len(raw) != ed25519.SignatureSize {
		return fmt.Errorf("signature for key %q must decode to %d bytes, got %d", signature.KeyID, ed25519.SignatureSize, len(raw))
	}
	if !ed25519.Verify(publicKey, message, raw) {
		return fmt.Errorf("signature for key %q does not verify", signature.KeyID)
	}
	return nil
}

// SignManifest 用全部 seed 对发布清单原始字节签名，返回按 key_id 升序的签名文件。
// 重复 seed（相同 key_id）被视为错误，拒绝生成歧义签名集合。
func SignManifest(raw []byte, seeds [][]byte) (*Signatures, error) {
	if len(seeds) == 0 {
		return nil, fmt.Errorf("at least one private key seed is required")
	}
	signatures := &Signatures{Schema: SignaturesSchema}
	seen := map[string]bool{}
	for _, seed := range seeds {
		signature, err := Sign(seed, raw)
		if err != nil {
			return nil, err
		}
		if seen[signature.KeyID] {
			return nil, fmt.Errorf("duplicate signing key %q", signature.KeyID)
		}
		seen[signature.KeyID] = true
		signatures.Signatures = append(signatures.Signatures, signature)
	}
	sort.Slice(signatures.Signatures, func(i, j int) bool {
		return signatures.Signatures[i].KeyID < signatures.Signatures[j].KeyID
	})
	return signatures, nil
}

// Canonical 返回签名文件的规范 JSON 字节。
func (s *Signatures) Canonical() ([]byte, error) {
	return canonicalJSON(s)
}

// ParseSignatures 严格解析并校验签名文件：拒绝重复键、非规范编码、空数组、
// 重复或无序 key_id、错误长度与非法编码的签名。
func ParseSignatures(raw []byte) (*Signatures, error) {
	var signatures Signatures
	if err := decodeStrictJSON(raw, &signatures); err != nil {
		return nil, fmt.Errorf("parse release signatures: %w", err)
	}
	canonical, err := signatures.Canonical()
	if err != nil {
		return nil, fmt.Errorf("canonicalize release signatures: %w", err)
	}
	if !bytes.Equal(canonical, raw) {
		return nil, fmt.Errorf("release signatures are not canonically encoded")
	}
	if signatures.Schema != SignaturesSchema {
		return nil, fmt.Errorf("unsupported release signatures schema %q", signatures.Schema)
	}
	if len(signatures.Signatures) == 0 {
		return nil, errors.New("release signatures file has no signatures")
	}
	var previous string
	for index, signature := range signatures.Signatures {
		if !isLowercaseSHA256Hex(signature.KeyID) {
			return nil, fmt.Errorf("release signature %d has invalid key_id %q", index, signature.KeyID)
		}
		if index > 0 && signature.KeyID <= previous {
			return nil, fmt.Errorf("release signature key_ids must be unique and sorted, found %q after %q", signature.KeyID, previous)
		}
		previous = signature.KeyID
		rawSignature, err := base64.StdEncoding.DecodeString(signature.Signature)
		if err != nil {
			return nil, fmt.Errorf("release signature %d has invalid base64: %w", index, err)
		}
		if len(rawSignature) != ed25519.SignatureSize {
			return nil, fmt.Errorf("release signature %d must decode to %d bytes, got %d", index, ed25519.SignatureSize, len(rawSignature))
		}
	}
	return &signatures, nil
}
