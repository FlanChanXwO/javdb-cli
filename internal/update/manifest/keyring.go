package manifest

import (
	"crypto/ed25519"
	"fmt"
)

// Keyring 是客户端内置的 key_id -> Ed25519 公钥环。
// 远端清单不能修改信任门槛；门槛固定为至少一份内置受信公钥验签成功。
type Keyring struct {
	keys map[string]ed25519.PublicKey
}

// NewKeyring 创建空公钥环。
func NewKeyring() *Keyring {
	return &Keyring{keys: map[string]ed25519.PublicKey{}}
}

// Add 注册受信公钥；相同 key_id 重复注册会被拒绝。
func (k *Keyring) Add(publicKey ed25519.PublicKey) error {
	if k == nil {
		return fmt.Errorf("keyring is nil")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("Ed25519 public key must be %d bytes", ed25519.PublicKeySize)
	}
	keyID := KeyID(publicKey)
	if _, exists := k.keys[keyID]; exists {
		return fmt.Errorf("trusted key %q is already registered", keyID)
	}
	k.keys[keyID] = publicKey
	return nil
}

// Has 报告 key_id 是否在受信环内。
func (k *Keyring) Has(keyID string) bool {
	if k == nil {
		return false
	}
	_, exists := k.keys[keyID]
	return exists
}

// VerifySignatures 校验签名文件中的受信签名；未知 key 可以与受信签名共存，
// 但不能单独通过。至少一份内置受信公钥验签成功才返回 nil。
func (k *Keyring) VerifySignatures(message []byte, signatures *Signatures) error {
	if k == nil || len(k.keys) == 0 {
		return fmt.Errorf("no trusted release keys are configured")
	}
	if signatures == nil {
		return fmt.Errorf("release signatures are nil")
	}
	for _, signature := range signatures.Signatures {
		publicKey, trusted := k.keys[signature.KeyID]
		if !trusted {
			continue
		}
		if err := Verify(message, signature, publicKey); err != nil {
			continue
		}
		return nil
	}
	return fmt.Errorf("release manifest is not signed by any trusted key")
}

// DefaultKeyring 返回生产发布签名的受信公钥环。
//
// 生产公钥在维护者按 runbook（docs/maintainers/development.md）生成密钥并把
// seed 放入 GitHub release environment secret 后登记。禁止提交测试之外任何
// 私钥；轮换时新旧公钥同时登记，过渡期清单由新旧私钥双签。
func DefaultKeyring() *Keyring {
	ring := NewKeyring()
	// 生产公钥（2026-08-13 首次发布登记；对应 seed 位于 GitHub release
	// environment 的 JAVDB_RELEASE_ED25519_PRIVATE_KEYS）。
	if err := ring.Add(productionKey0); err != nil {
		panic(err)
	}
	return ring
}

// productionKey0 是首个生产 Ed25519 公钥（key_id fffb9cd1...）。
var productionKey0 = ed25519.PublicKey{
	0x96, 0x93, 0x25, 0x97, 0xb8, 0xbb, 0x1e, 0x8d, 0x44, 0x3c, 0xcd, 0xa3, 0x05, 0x4f, 0x21, 0x25,
	0x2b, 0xf5, 0x83, 0x8d, 0x32, 0xb2, 0xaf, 0x8b, 0x37, 0x45, 0xd9, 0x63, 0x0e, 0x0e, 0x71, 0x45,
}
