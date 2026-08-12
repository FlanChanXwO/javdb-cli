package manifest

import (
	"testing"
)

func TestKeyringAddRejectsDuplicateKeyIDs(t *testing.T) {
	ring := NewKeyring()
	keyA := fixtureKey(fixtureSeedA)
	if err := ring.Add(keyA); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ring.Add(keyA); err == nil {
		t.Fatal("Keyring accepted the same public key twice")
	}
	if err := ring.Add(fixtureKey(fixtureSeedB)); err != nil {
		t.Fatalf("Add second key: %v", err)
	}
}

func TestKeyringRejectsMalformedPublicKey(t *testing.T) {
	ring := NewKeyring()
	if err := ring.Add(nil); err == nil {
		t.Fatal("Keyring accepted a nil public key")
	}
	if err := ring.Add(make([]byte, 31)); err == nil {
		t.Fatal("Keyring accepted a truncated public key")
	}
}

func TestVerifySignaturesAcceptsTrustedSignature(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	signatures, err := SignManifest(raw, [][]byte{fixtureSeedA, fixtureSeedB})
	if err != nil {
		t.Fatalf("SignManifest: %v", err)
	}
	canonical, err := signatures.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	parsed, err := ParseSignatures(canonical)
	if err != nil {
		t.Fatalf("ParseSignatures: %v", err)
	}

	// 双签文档：ring 里只有 A 或只有 B 都能通过（至少一份受信签名有效）。
	ringA := NewKeyring()
	if err := ringA.Add(fixtureKey(fixtureSeedA)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ringA.VerifySignatures(raw, parsed); err != nil {
		t.Errorf("VerifySignatures with key A: %v", err)
	}

	// 未知 key 可以与受信签名共存。
	ringAandC := NewKeyring()
	for _, seed := range [][]byte{fixtureSeedA, fixtureSeedC} {
		if err := ringAandC.Add(fixtureKey(seed)); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}
	if err := ringAandC.VerifySignatures(raw, parsed); err != nil {
		t.Errorf("VerifySignatures with trusted A plus unknown C: %v", err)
	}
}

func TestVerifySignaturesRejectsUntrusted(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	signatures, err := SignManifest(raw, [][]byte{fixtureSeedA, fixtureSeedB})
	if err != nil {
		t.Fatalf("SignManifest: %v", err)
	}
	canonical, err := signatures.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	parsed, err := ParseSignatures(canonical)
	if err != nil {
		t.Fatalf("ParseSignatures: %v", err)
	}

	// 未知 key 不能单独通过。
	ringOnlyUnknown := NewKeyring()
	if err := ringOnlyUnknown.Add(fixtureKey(fixtureSeedC)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ringOnlyUnknown.VerifySignatures(raw, parsed); err == nil {
		t.Error("VerifySignatures accepted signatures from unknown keys only")
	}

	// 空 ring 必须失败。
	if err := NewKeyring().VerifySignatures(raw, parsed); err == nil {
		t.Error("VerifySignatures accepted signatures with an empty keyring")
	}
}

func TestVerifySignaturesRejectsTamperedManifest(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	signatures, err := SignManifest(raw, [][]byte{fixtureSeedA})
	if err != nil {
		t.Fatalf("SignManifest: %v", err)
	}
	canonical, err := signatures.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	parsed, err := ParseSignatures(canonical)
	if err != nil {
		t.Fatalf("ParseSignatures: %v", err)
	}

	tampered := append([]byte{}, raw...)
	tampered[len(tampered)-1] ^= 0x01
	ring := NewKeyring()
	if err := ring.Add(fixtureKey(fixtureSeedA)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ring.VerifySignatures(tampered, parsed); err == nil {
		t.Error("VerifySignatures accepted tampered manifest bytes")
	}
}

func TestDefaultKeyringIsUsable(t *testing.T) {
	ring := DefaultKeyring()
	if ring == nil {
		t.Fatal("DefaultKeyring returned nil")
	}
	// 当前仓库还没有生产公钥；ring 允许后续运行 runbook 添加首个密钥。
	if err := ring.Add(fixtureKey(fixtureSeedC)); err != nil {
		t.Errorf("DefaultKeyring cannot be extended: %v", err)
	}
}
