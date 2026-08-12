package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/update/manifest"
)

// TEST-ONLY fixture Ed25519 seeds.
//
//	这些密钥只用于单元测试，绝不用于生产签名；它们是与测试代码一起提交的固定
//	常量，任何拿到源码的人都能重建对应私钥。生产签名只允许使用
//	JAVDB_RELEASE_ED25519_PRIVATE_KEYS 环境变量注入的密钥。
var (
	fixtureSeedA = bytes.Repeat([]byte{0x01}, ed25519.SeedSize)
	fixtureSeedB = bytes.Repeat([]byte{0x02}, ed25519.SeedSize)
)

const testReleaseDate = "2026-08-12"

func fixtureKey(t *testing.T, seed []byte) ed25519.PublicKey {
	t.Helper()
	return ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
}

func fixtureKeyring(t *testing.T, seeds ...[]byte) *manifest.Keyring {
	t.Helper()
	ring := manifest.NewKeyring()
	for _, seed := range seeds {
		if err := ring.Add(fixtureKey(t, seed)); err != nil {
			t.Fatalf("Add fixture key: %v", err)
		}
	}
	return ring
}

// signedManifestFor 构造当前平台使用真实哈希、其余平台使用占位哈希的签名清单。
func signedManifestFor(t *testing.T, tag, goos, goarch, archiveName string, archive []byte, binary []byte) ([]byte, []byte) {
	t.Helper()
	version := strings.TrimPrefix(tag, "v")
	archiveSum := sha256.Sum256(archive)
	binarySum := sha256.Sum256(binary)
	targets := make([]manifest.Target, 0, 6)
	for _, platform := range []struct{ goos, goarch string }{
		{goos: "darwin", goarch: "amd64"},
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "linux", goarch: "arm64"},
		{goos: "windows", goarch: "amd64"},
		{goos: "windows", goarch: "arm64"},
	} {
		extension := ".tar.gz"
		binaryName := "javdb"
		if platform.goos == "windows" {
			extension = ".zip"
			binaryName = "javdb.exe"
		}
		target := manifest.Target{
			GOOS:          platform.goos,
			GOARCH:        platform.goarch,
			Archive:       "javdb-cli_" + version + "_" + platform.goos + "_" + platform.goarch + extension,
			ArchiveSHA256: strings.Repeat("a", 64),
			Binary:        binaryName,
			BinarySHA256:  strings.Repeat("b", 64),
		}
		if platform.goos == goos && platform.goarch == goarch {
			target.ArchiveSHA256 = hex.EncodeToString(archiveSum[:])
			target.BinarySHA256 = hex.EncodeToString(binarySum[:])
		}
		targets = append(targets, target)
	}
	releaseManifest, err := manifest.NewManifest(tag, testReleaseDate, targets)
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	manifestBytes, err := releaseManifest.Canonical()
	if err != nil {
		t.Fatalf("Canonical manifest: %v", err)
	}
	signatures, err := manifest.SignManifest(manifestBytes, [][]byte{fixtureSeedA})
	if err != nil {
		t.Fatalf("SignManifest: %v", err)
	}
	signatureBytes, err := signatures.Canonical()
	if err != nil {
		t.Fatalf("Canonical signatures: %v", err)
	}
	_ = archiveName
	return manifestBytes, signatureBytes
}

// installReleaseAssets 用本地 server 提供 manifest、signatures 与 archive，
// 执行完整安装流程并返回错误与目标二进制路径。
func installReleaseAssets(t *testing.T, goos, goarch, archiveName string, archive []byte, manifestBytes, signatureBytes []byte, keyring *manifest.Keyring) (error, string) {
	t.Helper()
	target := filepath.Join(t.TempDir(), binaryNameFor(goos))
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           goos,
		GOARCH:         goarch,
		Keyring:        keyring,
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, archiveName)})
	return err, target
}

func binaryNameFor(goos string) string {
	if goos == "windows" {
		return "javdb.exe"
	}
	return "javdb"
}

func releaseAssets(serverURL, archiveName string) []ReleaseAsset {
	return []ReleaseAsset{
		{Name: manifestAssetName, DownloadURL: serverURL + "/manifest"},
		{Name: signaturesAssetName, DownloadURL: serverURL + "/signatures"},
		{Name: archiveName, DownloadURL: serverURL + "/archive"},
	}
}

func TestReleaseInstallerVerifiesAndReplacesTarBinary(t *testing.T) {
	// 归档内容是不可执行文本；候选二进制绝不执行，只有哈希和签名验证。
	archive := tarGzArchive(t, "javdb", []byte("verified linux binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("verified linux binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	if err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_linux_amd64.tar.gz")}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "verified linux binary" {
		t.Fatalf("installed binary = %q", body)
	}
}

func TestReleaseInstallerVerifiesAndReplacesZIPBinary(t *testing.T) {
	archive := zipArchive(t, "javdb.exe", []byte("verified windows binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "windows", "arm64", "javdb-cli_0.2.0_windows_arm64.zip", archive, []byte("verified windows binary"))
	target := filepath.Join(t.TempDir(), "javdb.exe")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "windows",
		GOARCH:         "arm64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	if err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_windows_arm64.zip")}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "verified windows binary" {
		t.Fatalf("installed binary = %q", body)
	}
}

// assertKeepsCurrentBinary 断言安装失败后现有二进制保持不变（零写入语义）。
func assertKeepsCurrentBinary(t *testing.T, target string, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("install unexpectedly succeeded")
	}
	body, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(body) != "old binary" {
		t.Fatalf("target changed to %q after failed install", body)
	}
}

func TestReleaseInstallerRejectsUntrustedSignature(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("untrusted binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("untrusted binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	// 签名由 fixture key A 生成，但 ring 只信任 key B。
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedB),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_linux_amd64.tar.gz")})
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsTamperedManifest(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("tampered binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("tampered binary"))
	// 篡改清单字节（翻转 version 数字），签名不再匹配。
	flipped := append([]byte{}, manifestBytes...)
	for index := len(flipped) - 1; index >= 0; index-- {
		if flipped[index] == '0' {
			flipped[index] = '9'
			break
		}
	}
	err, target := installReleaseAssets(t, "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, flipped, signatureBytes, fixtureKeyring(t, fixtureSeedA))
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsReplayedManifest(t *testing.T) {
	// 重放攻击：清单 tag 是 v0.1.0，但 Release 是 v0.2.0。
	archive := tarGzArchive(t, "javdb", []byte("replayed binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.1.0", "linux", "amd64", "javdb-cli_0.1.0_linux_amd64.tar.gz", archive, []byte("replayed binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(archive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	// Release 选择 v0.2.0，但 manifest 绑定 v0.1.0。
	err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_linux_amd64.tar.gz")})
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsWrongPlatform(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("platform binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("platform binary"))
	// 运行平台 plan9/amd64 不在清单六平台集合内。
	err, target := installReleaseAssets(t, "plan9", "amd64", "javdb-cli_0.2.0_plan9_amd64.tar.gz", archive, manifestBytes, signatureBytes, fixtureKeyring(t, fixtureSeedA))
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsNonOfficialAssetURL(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("fake url binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("fake url binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 使用默认 URL validator；URL 指向本地 server 而非官方 GitHub 地址。
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     http.DefaultClient,
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
	})
	release := Release{TagName: "v0.2.0", Assets: []ReleaseAsset{
		{Name: manifestAssetName, DownloadURL: "https://evil.example/manifest"},
		{Name: signaturesAssetName, DownloadURL: "https://evil.example/signatures"},
		{Name: "javdb-cli_0.2.0_linux_amd64.tar.gz", DownloadURL: "https://evil.example/archive"},
	}}
	err := installer.Install(context.Background(), release)
	if err == nil {
		t.Fatal("install accepted non-official asset URLs")
	}
	body, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(body) != "old binary" {
		t.Fatalf("target changed to %q", body)
	}
	_ = manifestBytes
	_ = signatureBytes
}

func TestReleaseInstallerRejectsDuplicateOrMissingAssets(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("asset binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("asset binary"))
	_ = manifestBytes
	_ = signatureBytes

	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	duplicate := Release{TagName: "v0.2.0", Assets: []ReleaseAsset{
		{Name: "javdb-cli_0.2.0_linux_amd64.tar.gz", DownloadURL: "http://localhost/archive"},
		{Name: "javdb-cli_0.2.0_linux_amd64.tar.gz", DownloadURL: "http://localhost/archive2"},
		{Name: manifestAssetName, DownloadURL: "http://localhost/manifest"},
		{Name: signaturesAssetName, DownloadURL: "http://localhost/signatures"},
	}}
	if err := installer.Install(context.Background(), duplicate); err == nil {
		t.Fatal("install accepted duplicate archive assets")
	}
	missing := Release{TagName: "v0.2.0", Assets: []ReleaseAsset{
		{Name: "javdb-cli_0.2.0_linux_amd64.tar.gz", DownloadURL: "http://localhost/archive"},
		{Name: signaturesAssetName, DownloadURL: "http://localhost/signatures"},
	}}
	if err := installer.Install(context.Background(), missing); err == nil {
		t.Fatal("install accepted a release without the manifest asset")
	}
	body, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(body) != "old binary" {
		t.Fatalf("target changed to %q", body)
	}
}

func TestReleaseInstallerRejectsArchiveHashMismatch(t *testing.T) {
	// 清单为 archive 声明了哈希，但 server 返回不同字节。
	expectedArchive := tarGzArchive(t, "javdb", []byte("expected binary"))
	actualArchive := tarGzArchive(t, "javdb", []byte("actual binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", expectedArchive, []byte("expected binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(actualArchive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_linux_amd64.tar.gz")})
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsBinaryHashMismatch(t *testing.T) {
	// archive 与清单一致，但内部二进制字节被替换。
	expectedArchive := tarGzArchive(t, "javdb", []byte("expected binary"))
	swappedArchive := tarGzArchive(t, "javdb", []byte("swapped binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", expectedArchive, []byte("expected binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(swappedArchive)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_linux_amd64.tar.gz")})
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsExtractionErrors(t *testing.T) {
	// 归档不包含预期二进制条目。
	var archive bytes.Buffer
	gzipped := gzip.NewWriter(&archive)
	writer := tar.NewWriter(gzipped)
	if err := writer.WriteHeader(&tar.Header{Name: "README.md", Mode: 0o644, Size: int64(len("readme"))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("readme")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipped.Close(); err != nil {
		t.Fatal(err)
	}
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive.Bytes(), []byte("expected binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/manifest":
			_, _ = writer.Write(manifestBytes)
		case "/signatures":
			_, _ = writer.Write(signatureBytes)
		case "/archive":
			_, _ = writer.Write(archive.Bytes())
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	installer := NewReleaseInstaller(ReleaseInstallerOptions{
		HTTPClient:     server.Client(),
		ExecutablePath: func() (string, error) { return target, nil },
		GOOS:           "linux",
		GOARCH:         "amd64",
		Keyring:        fixtureKeyring(t, fixtureSeedA),
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	err := installer.Install(context.Background(), Release{TagName: "v0.2.0", Assets: releaseAssets(server.URL, "javdb-cli_0.2.0_linux_amd64.tar.gz")})
	assertKeepsCurrentBinary(t, target, err)
}

func TestReleaseInstallerRejectsEmptyTrustedKeyring(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("empty ring binary"))
	manifestBytes, signatureBytes := signedManifestFor(t, "v0.2.0", "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, []byte("empty ring binary"))
	err, target := installReleaseAssets(t, "linux", "amd64", "javdb-cli_0.2.0_linux_amd64.tar.gz", archive, manifestBytes, signatureBytes, manifest.NewKeyring())
	assertKeepsCurrentBinary(t, target, err)
}

func tarGzArchive(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipped := gzip.NewWriter(&archive)
	writer := tar.NewWriter(gzipped)
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipped.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func zipArchive(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
