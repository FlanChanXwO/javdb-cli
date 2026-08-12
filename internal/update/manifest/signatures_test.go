package manifest

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func TestKeyIDDerivationIsStableAndUnique(t *testing.T) {
	keyA := fixtureKey(fixtureSeedA)
	keyB := fixtureKey(fixtureSeedB)

	first := KeyID(keyA)
	second := KeyID(keyA)
	if first != second {
		t.Errorf("KeyID is not deterministic: %q != %q", first, second)
	}
	if len(first) != 64 {
		t.Errorf("KeyID length = %d, want 64 hex chars", len(first))
	}
	if strings.ToLower(first) != first {
		t.Error("KeyID must be lowercase hex")
	}
	if _, err := hex.DecodeString(first); err != nil {
		t.Errorf("KeyID is not valid hex: %v", err)
	}
	if KeyID(keyB) == first {
		t.Error("distinct keys must produce distinct key IDs")
	}
}

func TestSignAndVerifySingleKey(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	signature, err := Sign(fixtureSeedA, raw)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if signature.KeyID != KeyID(fixtureKey(fixtureSeedA)) {
		t.Errorf("signature key_id = %q, want derived from signer key", signature.KeyID)
	}

	if err := Verify(raw, signature, fixtureKey(fixtureSeedA)); err != nil {
		t.Errorf("Verify with correct key: %v", err)
	}
	if err := Verify(raw, signature, fixtureKey(fixtureSeedB)); err == nil {
		t.Error("Verify accepted signature from the wrong key")
	}

	tampered := append([]byte{}, raw...)
	tampered[len(tampered)-1] ^= 0x01
	if err := Verify(tampered, signature, fixtureKey(fixtureSeedA)); err == nil {
		t.Error("Verify accepted signature over tampered message")
	}
}

func TestVerifyRejectsMalformedSignatures(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	key := fixtureKey(fixtureSeedA)

	short := Signature{KeyID: KeyID(key), Signature: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize-1))}
	if err := Verify(raw, short, key); err == nil {
		t.Error("Verify accepted truncated signature")
	}
	long := Signature{KeyID: KeyID(key), Signature: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize+1))}
	if err := Verify(raw, long, key); err == nil {
		t.Error("Verify accepted oversized signature")
	}
	badBase64 := Signature{KeyID: KeyID(key), Signature: "!!!not-base64!!!"}
	if err := Verify(raw, badBase64, key); err == nil {
		t.Error("Verify accepted invalid base64 signature")
	}
	if err := Verify(raw, Signature{KeyID: KeyID(key)}, key); err == nil {
		t.Error("Verify accepted empty signature")
	}
}

func TestSignManifestWithSingleKey(t *testing.T) {
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
	if signatures.Schema != SignaturesSchema {
		t.Errorf("schema = %q, want %q", signatures.Schema, SignaturesSchema)
	}
	if len(signatures.Signatures) != 1 {
		t.Fatalf("signature count = %d, want 1", len(signatures.Signatures))
	}
	canonical, err := signatures.Canonical()
	if err != nil {
		t.Fatalf("Signatures.Canonical: %v", err)
	}
	parsed, err := ParseSignatures(canonical)
	if err != nil {
		t.Fatalf("ParseSignatures: %v", err)
	}
	if parsed.Signatures[0].KeyID != KeyID(fixtureKey(fixtureSeedA)) {
		t.Errorf("key_id = %q, want fixture key A", parsed.Signatures[0].KeyID)
	}
}

func TestSignManifestRotationDoubleSignature(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	signatures, err := SignManifest(raw, [][]byte{fixtureSeedB, fixtureSeedA})
	if err != nil {
		t.Fatalf("SignManifest: %v", err)
	}
	if len(signatures.Signatures) != 2 {
		t.Fatalf("signature count = %d, want 2", len(signatures.Signatures))
	}
	// 签名必须按 key_id 升序排列。
	if signatures.Signatures[0].KeyID > signatures.Signatures[1].KeyID {
		t.Error("signatures are not sorted by key_id")
	}
	// 每个签名必须能被其声明的 key_id 对应公钥验证。
	byID := map[string]Signature{}
	for _, signature := range signatures.Signatures {
		if _, exists := byID[signature.KeyID]; exists {
			t.Fatalf("duplicate signature key_id %q", signature.KeyID)
		}
		byID[signature.KeyID] = signature
	}
	keyA, keyB := fixtureKey(fixtureSeedA), fixtureKey(fixtureSeedB)
	for keyID, signature := range byID {
		key := keyA
		if keyID != KeyID(keyA) {
			key = keyB
		}
		if err := Verify(raw, signature, key); err != nil {
			t.Errorf("signature %q does not verify with its declared key: %v", keyID, err)
		}
	}
}

func TestSignManifestRejectsDuplicateSeeds(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	if _, err := SignManifest(raw, [][]byte{fixtureSeedA, fixtureSeedA}); err == nil {
		t.Fatal("SignManifest accepted duplicate seeds")
	}
}

func TestParseSignaturesRejectsInvalidDocuments(t *testing.T) {
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

	// 复制第一个签名对象，造成重复 key_id。
	firstObject := bytes.Index(canonical, []byte(`{"key_id":`))
	if firstObject < 0 {
		t.Fatal("cannot locate first signature object")
	}
	firstEnd := bytes.Index(canonical[firstObject:], []byte(`}`)) + firstObject + 1
	duplicate := make([]byte, 0, len(canonical)+len(canonical))
	duplicate = append(duplicate, canonical[:firstObject]...)
	duplicate = append(duplicate, canonical[firstObject:firstEnd]...)
	duplicate = append(duplicate, ',')
	duplicate = append(duplicate, canonical[firstObject:]...)

	wrongSchema := mutateField(t, canonical, "schema", `"javdb.release-signatures/v9"`)
	emptyArray := mutateField(t, canonical, "signatures", `[]`)
	badBase64 := mutateField(t, canonical, "signature", `"!!!not-base64!!!"`)
	shortSignature := mutateField(t, canonical, "signature", `"`+base64.StdEncoding.EncodeToString(make([]byte, 31))+`"`)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, canonical, "", "  "); err != nil {
		t.Fatalf("indent: %v", err)
	}

	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{name: "duplicate key_id", raw: duplicate},
		{name: "wrong schema", raw: wrongSchema},
		{name: "empty signatures array", raw: emptyArray},
		{name: "invalid base64 signature", raw: badBase64},
		{name: "short signature bytes", raw: shortSignature},
		{name: "non canonical encoding", raw: pretty.Bytes()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseSignatures(tc.raw); err == nil {
				t.Fatal("ParseSignatures accepted invalid document")
			}
		})
	}
}

func TestParseSignaturesRejectsUnsortedKeyIDs(t *testing.T) {
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
	if len(signatures.Signatures) != 2 {
		t.Fatalf("want two signatures, got %d", len(signatures.Signatures))
	}
	// 手工构造 key_id 逆序的签名文件。
	swapped := Signatures{
		Schema:     SignaturesSchema,
		Signatures: []Signature{signatures.Signatures[1], signatures.Signatures[0]},
	}
	swappedBytes, err := swapped.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	if _, err := ParseSignatures(swappedBytes); err == nil {
		t.Fatal("ParseSignatures accepted unsorted key_ids")
	}
}

func TestParseSignaturesRejectsOversizedSignature(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := manifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	valid, err := SignManifest(raw, [][]byte{fixtureSeedA})
	if err != nil {
		t.Fatalf("SignManifest: %v", err)
	}
	valid.Signatures[0].Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize+4))
	canonical, err := valid.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	if _, err := ParseSignatures(canonical); err == nil {
		t.Fatal("ParseSignatures accepted oversized signature")
	}
}
