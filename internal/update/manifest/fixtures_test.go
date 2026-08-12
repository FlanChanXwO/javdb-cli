package manifest

import (
	"bytes"
	"crypto/ed25519"
	"strings"
)

// TEST-ONLY fixture Ed25519 seeds.
//
//	这些密钥只用于单元测试，绝不用于生产签名；它们是与测试代码一起提交的固定
//	常量，任何拿到源码的人都能重建对应私钥。生产签名只允许使用
//	JAVDB_RELEASE_ED25519_PRIVATE_KEYS 环境变量注入的密钥。
var (
	fixtureSeedA = bytes.Repeat([]byte{0x01}, ed25519.SeedSize)
	fixtureSeedB = bytes.Repeat([]byte{0x02}, ed25519.SeedSize)
	fixtureSeedC = bytes.Repeat([]byte{0x03}, ed25519.SeedSize)
)

// fixtureKey 从测试专用 seed 派生公钥，测试断言签名与 keyring 行为时使用。
func fixtureKey(seed []byte) ed25519.PublicKey {
	return ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
}

func validTargets() []Target {
	return []Target{
		{GOOS: "darwin", GOARCH: "amd64", Archive: "javdb-cli_0.7.0_darwin_amd64.tar.gz", ArchiveSHA256: strings.Repeat("a", 64), Binary: "javdb", BinarySHA256: strings.Repeat("b", 64)},
		{GOOS: "darwin", GOARCH: "arm64", Archive: "javdb-cli_0.7.0_darwin_arm64.tar.gz", ArchiveSHA256: strings.Repeat("c", 64), Binary: "javdb", BinarySHA256: strings.Repeat("d", 64)},
		{GOOS: "linux", GOARCH: "amd64", Archive: "javdb-cli_0.7.0_linux_amd64.tar.gz", ArchiveSHA256: strings.Repeat("e", 64), Binary: "javdb", BinarySHA256: strings.Repeat("f", 64)},
		{GOOS: "linux", GOARCH: "arm64", Archive: "javdb-cli_0.7.0_linux_arm64.tar.gz", ArchiveSHA256: strings.Repeat("0", 64), Binary: "javdb", BinarySHA256: strings.Repeat("1", 64)},
		{GOOS: "windows", GOARCH: "amd64", Archive: "javdb-cli_0.7.0_windows_amd64.zip", ArchiveSHA256: strings.Repeat("2", 64), Binary: "javdb.exe", BinarySHA256: strings.Repeat("3", 64)},
		{GOOS: "windows", GOARCH: "arm64", Archive: "javdb-cli_0.7.0_windows_arm64.zip", ArchiveSHA256: strings.Repeat("4", 64), Binary: "javdb.exe", BinarySHA256: strings.Repeat("5", 64)},
	}
}
