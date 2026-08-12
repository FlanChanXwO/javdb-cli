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
// 当前仓库尚未产生生产密钥；首个公钥在维护者按 runbook 生成密钥并把 seed
// 放入 GitHub release environment secret 后由本文件登记。空环保持 fail-closed，
// 任何远程清单都不能通过验证。
func DefaultKeyring() *Keyring {
	// 生产公钥登记示例（禁止提交测试之外任何私钥；runbook 见
	// docs/maintainers/development.md）：
	//
	//	ring := NewKeyring()
	//	if err := ring.Add(productionPublicKey); err != nil { ... }
	//	return ring
	//
	// 添加首个生产公钥前，release 验证将显式失败（fail-closed）。
	return NewKeyring()
}
