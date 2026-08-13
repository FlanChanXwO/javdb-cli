package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/update/archive"
	"github.com/FlanChanXwO/javdb-cli/internal/update/manifest"
)

// TEST-ONLY fixture Ed25519 seeds.
//
//	这些密钥只用于单元测试，绝不用于生产签名；它们是与测试代码一起提交的固定
//	常量，任何拿到源码的人都能重建对应私钥。生产签名只允许使用
//	JAVDB_RELEASE_ED25519_PRIVATE_KEYS 环境变量注入的密钥。
var (
	fixtureSeedA = bytes.Repeat([]byte{0x01}, 32)
	fixtureSeedB = bytes.Repeat([]byte{0x02}, 32)
)

func fixtureSeedsJSON(t *testing.T, seeds [][]byte) string {
	t.Helper()
	encoded := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		encoded = append(encoded, base64.StdEncoding.EncodeToString(seed))
	}
	raw, err := json.Marshal(encoded)
	if err != nil {
		t.Fatalf("marshal seeds: %v", err)
	}
	return string(raw)
}

func writeReleaseDir(t *testing.T, version string) string {
	t.Helper()
	dir := t.TempDir()
	for _, platform := range releasePlatforms {
		extension := ".tar.gz"
		binaryName := "javdb"
		binaryContent := []byte("fixture binary " + platform.goos + "/" + platform.goarch)
		if platform.goos == "windows" {
			extension = ".zip"
			binaryName = "javdb.exe"
		}
		name := "javdb-cli_" + version + "_" + platform.goos + "_" + platform.goarch + extension
		path := filepath.Join(dir, name)
		var content []byte
		var err error
		if extension == ".zip" {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			entry, entryErr := writer.Create(binaryName)
			if entryErr != nil {
				t.Fatalf("create zip entry: %v", entryErr)
			}
			if _, entryErr = entry.Write(binaryContent); entryErr != nil {
				t.Fatalf("write zip entry: %v", entryErr)
			}
			if entryErr = writer.Close(); entryErr != nil {
				t.Fatalf("close zip: %v", entryErr)
			}
			content = buffer.Bytes()
		} else {
			var buffer bytes.Buffer
			gzipped := gzip.NewWriter(&buffer)
			tarWriter := tar.NewWriter(gzipped)
			header := &tar.Header{Name: binaryName, Mode: 0o755, Size: int64(len(binaryContent))}
			if err = tarWriter.WriteHeader(header); err != nil {
				t.Fatalf("write tar header: %v", err)
			}
			if _, err = tarWriter.Write(binaryContent); err != nil {
				t.Fatalf("write tar entry: %v", err)
			}
			if err = tarWriter.Close(); err != nil {
				t.Fatalf("close tar: %v", err)
			}
			if err = gzipped.Close(); err != nil {
				t.Fatalf("close gzip: %v", err)
			}
			content = buffer.Bytes()
		}
		if err = os.WriteFile(path, content, 0o644); err != nil {
			t.Fatalf("write archive: %v", err)
		}
	}
	return dir
}

func TestRunGenerateProducesVerifiedManifestAndSignatures(t *testing.T) {
	const version = "0.7.0"
	const date = "2026-08-12"
	dir := writeReleaseDir(t, version)

	var output bytes.Buffer
	err := runGenerate(generateOptions{
		version:     version,
		releaseDate: date,
		dir:         dir,
		outputDir:   dir,
		getenv: func(name string) string {
			if name == privateKeysEnvironment {
				return fixtureSeedsJSON(t, [][]byte{fixtureSeedA, fixtureSeedB})
			}
			return ""
		},
		stdout: &output,
	})
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil {
		t.Fatalf("read release-manifest.json: %v", err)
	}
	releaseManifest, err := manifest.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if releaseManifest.Tag != "v"+version || releaseManifest.ReleaseDate != date {
		t.Errorf("manifest metadata = %s %s", releaseManifest.Tag, releaseManifest.ReleaseDate)
	}
	if len(releaseManifest.Targets) != 6 {
		t.Fatalf("target count = %d, want 6", len(releaseManifest.Targets))
	}

	for _, target := range releaseManifest.Targets {
		archivePath := filepath.Join(dir, target.Archive)
		archiveContent, err := os.ReadFile(archivePath)
		if err != nil {
			t.Fatalf("read archive %q: %v", target.Archive, err)
		}
		archiveSum := sha256.Sum256(archiveContent)
		if target.ArchiveSHA256 != hex.EncodeToString(archiveSum[:]) {
			t.Errorf("archive hash mismatch for %s", target.Archive)
		}
	}

	signatureBytes, err := os.ReadFile(filepath.Join(dir, "release-manifest.sig"))
	if err != nil {
		t.Fatalf("read release-manifest.sig: %v", err)
	}
	signatures, err := manifest.ParseSignatures(signatureBytes)
	if err != nil {
		t.Fatalf("ParseSignatures: %v", err)
	}
	if len(signatures.Signatures) != 2 {
		t.Fatalf("signature count = %d, want 2", len(signatures.Signatures))
	}

	ring := manifest.NewKeyring()
	if err := ring.Add(ed25519PublicKey(t, fixtureSeedA)); err != nil {
		t.Fatalf("Add key A: %v", err)
	}
	if err := ring.Add(ed25519PublicKey(t, fixtureSeedB)); err != nil {
		t.Fatalf("Add key B: %v", err)
	}
	if err := ring.VerifySignatures(manifestBytes, signatures); err != nil {
		t.Errorf("VerifySignatures: %v", err)
	}
}

func ed25519PublicKey(t *testing.T, seed []byte) ed25519.PublicKey {
	t.Helper()
	return ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
}

func TestRunGenerateWritesBinaryHashFromArchiveContent(t *testing.T) {
	const version = "0.7.0"
	dir := writeReleaseDir(t, version)

	err := runGenerate(generateOptions{
		version:     version,
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		getenv: func(string) string {
			return fixtureSeedsJSON(t, [][]byte{fixtureSeedA})
		},
		stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	releaseManifest, err := manifest.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	for _, target := range releaseManifest.Targets {
		expected := sha256.Sum256([]byte("fixture binary " + target.GOOS + "/" + target.GOARCH))
		if target.BinarySHA256 != hex.EncodeToString(expected[:]) {
			t.Errorf("binary hash mismatch for %s/%s", target.GOOS, target.GOARCH)
		}
	}
}

func TestRunGenerateRejectsMissingSeedEnvironment(t *testing.T) {
	dir := writeReleaseDir(t, "0.7.0")
	err := runGenerate(generateOptions{
		version:     "0.7.0",
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		getenv:      func(string) string { return "" },
		stdout:      io.Discard,
	})
	if err == nil {
		t.Fatal("runGenerate accepted missing private key environment")
	}
	if strings.Contains(err.Error(), "0x01") || strings.Contains(err.Error(), base64.StdEncoding.EncodeToString(fixtureSeedA)) {
		t.Error("error message leaks seed material")
	}
}

func TestRunGenerateRejectsInvalidSeedValues(t *testing.T) {
	dir := writeReleaseDir(t, "0.7.0")
	for _, tc := range []struct {
		name string
		env  string
	}{
		{name: "not a JSON array", env: "not-json"},
		{name: "empty array", env: "[]"},
		{name: "invalid base64", env: `["@@@"]`},
		{name: "short seed", env: `["` + base64.StdEncoding.EncodeToString([]byte("short")) + `"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runGenerate(generateOptions{
				version:     "0.7.0",
				releaseDate: "2026-08-12",
				dir:         dir,
				outputDir:   dir,
				getenv: func(string) string {
					return tc.env
				},
				stdout: io.Discard,
			})
			if err == nil {
				t.Fatal("runGenerate accepted invalid seed environment")
			}
		})
	}
}

func TestRunGenerateRejectsDuplicateSeeds(t *testing.T) {
	dir := writeReleaseDir(t, "0.7.0")
	err := runGenerate(generateOptions{
		version:     "0.7.0",
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		getenv: func(string) string {
			return fixtureSeedsJSON(t, [][]byte{fixtureSeedA, fixtureSeedA})
		},
		stdout: io.Discard,
	})
	if err == nil {
		t.Fatal("runGenerate accepted duplicate seeds")
	}
}

func TestRunGenerateRejectsMissingArchive(t *testing.T) {
	dir := writeReleaseDir(t, "0.7.0")
	// 删除一个平台归档，模拟不完整资产集合。
	missing := filepath.Join(dir, "javdb-cli_0.7.0_windows_arm64.zip")
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove archive: %v", err)
	}
	err := runGenerate(generateOptions{
		version:     "0.7.0",
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		getenv: func(string) string {
			return fixtureSeedsJSON(t, [][]byte{fixtureSeedA})
		},
		stdout: io.Discard,
	})
	if err == nil {
		t.Fatal("runGenerate accepted an incomplete platform archive set")
	}
	if !strings.Contains(err.Error(), "windows_arm64") {
		t.Errorf("error should name the missing archive, got: %v", err)
	}
}

func TestRunGenerateRejectsExistingOutput(t *testing.T) {
	dir := writeReleaseDir(t, "0.7.0")
	options := generateOptions{
		version:     "0.7.0",
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		getenv: func(string) string {
			return fixtureSeedsJSON(t, [][]byte{fixtureSeedA})
		},
		stdout: io.Discard,
	}
	if err := runGenerate(options); err != nil {
		t.Fatalf("first runGenerate: %v", err)
	}
	if err := runGenerate(options); err == nil {
		t.Fatal("runGenerate overwrote existing output files")
	}
}

func TestRunGenerateRejectsMissingArguments(t *testing.T) {
	dir := writeReleaseDir(t, "0.7.0")
	for _, tc := range []struct {
		name        string
		version     string
		releaseDate string
	}{
		{name: "missing version", version: "", releaseDate: "2026-08-12"},
		{name: "missing release date", version: "0.7.0", releaseDate: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := runGenerate(generateOptions{
				version:     tc.version,
				releaseDate: tc.releaseDate,
				dir:         dir,
				outputDir:   dir,
				getenv: func(string) string {
					return fixtureSeedsJSON(t, [][]byte{fixtureSeedA})
				},
				stdout: io.Discard,
			})
			if err == nil {
				t.Fatal("runGenerate accepted missing arguments")
			}
		})
	}
}

// TestRunShowKeysPrintsOnlyPublicMaterial 验证 --show-keys 只输出公钥与
// key_id，绝不泄漏 seed/私钥字节。
func TestRunShowKeysPrintsOnlyPublicMaterial(t *testing.T) {
	var output bytes.Buffer
	err := runShowKeys(fixtureSeedsJSON(t, [][]byte{fixtureSeedA}), &output)
	if err != nil {
		t.Fatalf("runShowKeys: %v", err)
	}
	text := output.String()
	seedB64 := base64.StdEncoding.EncodeToString(fixtureSeedA)
	if strings.Contains(text, seedB64) {
		t.Fatal("runShowKeys printed the seed value")
	}
	expectedKeyID := manifest.KeyID(ed25519.NewKeyFromSeed(fixtureSeedA).Public().(ed25519.PublicKey))
	if !strings.Contains(text, "key_id="+expectedKeyID) {
		t.Errorf("runShowKeys output lacks the derived key_id: %s", text)
	}
	if !strings.Contains(text, "public_key_hex=") {
		t.Errorf("runShowKeys output lacks the public key: %s", text)
	}
}

func TestRunShowKeysRejectsInvalidEnvironment(t *testing.T) {
	var output bytes.Buffer
	if err := runShowKeys("not-json", &output); err == nil {
		t.Fatal("runShowKeys accepted an invalid seed environment")
	}
	if output.Len() != 0 {
		t.Error("runShowKeys wrote output despite invalid environment")
	}
}

// legacyChecksumLookup 复刻 v0.6.0 更新器的 checksums.txt 解析器语义：
// 每行恰好两个字段、哈希为 64 位小写十六进制、重复条目报错。
func legacyChecksumLookup(t *testing.T, checksums []byte, archiveName string) string {
	t.Helper()
	var expected string
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != archiveName {
			continue
		}
		if expected != "" {
			t.Fatalf("legacy updater would reject duplicate entry for %q", archiveName)
		}
		if len(fields[0]) != 64 || strings.ToLower(fields[0]) != fields[0] {
			t.Fatalf("legacy updater would reject invalid SHA-256 %q", fields[0])
		}
		expected = fields[0]
	}
	return expected
}

// TestRunGenerateChecksumsSatisfyLegacyUpdaterContract 是 v0.6.0 -> v0.6.1
// fixture 测试：v0.6.0 更新器只信任 checksums.txt + archive，本测试用旧解析
// 语义验证由清单派生的 checksums.txt 每个归档恰好一条且哈希匹配。
func TestRunGenerateChecksumsSatisfyLegacyUpdaterContract(t *testing.T) {
	const version = "0.7.0"
	dir := writeReleaseDir(t, version)
	checksumsPath := filepath.Join(dir, "checksums.txt")
	err := runGenerate(generateOptions{
		version:     version,
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		checksums:   checksumsPath,
		getenv: func(string) string {
			return fixtureSeedsJSON(t, [][]byte{fixtureSeedA})
		},
		stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}
	checksums, err := os.ReadFile(checksumsPath)
	if err != nil {
		t.Fatalf("read checksums.txt: %v", err)
	}
	for _, platform := range releasePlatforms {
		extension := ".tar.gz"
		if platform.goos == "windows" {
			extension = ".zip"
		}
		name := "javdb-cli_" + version + "_" + platform.goos + "_" + platform.goarch + extension
		expected, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read archive %q: %v", name, err)
		}
		sum := sha256.Sum256(expected)
		lookedUp := legacyChecksumLookup(t, checksums, name)
		if lookedUp == "" {
			t.Errorf("checksums.txt has no entry for %q", name)
			continue
		}
		if lookedUp != hex.EncodeToString(sum[:]) {
			t.Errorf("checksums.txt entry for %q does not match archive bytes", name)
		}
	}
}

// TestReleaseAssetFixtureSatisfiesLegacyAndSignedContracts 是三条升级路径的
// 资产级 fixture 测试：
//   - v0.6.0 → 更新：只需 checksums.txt + archive（旧契约，由清单派生）。
//   - v0.6.1/v0.7 → 更新：manifest 验签 + 归档/解包二进制双哈希（新契约）。
//
// 同一资产集必须同时满足两者，保证 v0.6.0 可直接验证并安装后续版本。
func TestReleaseAssetFixtureSatisfiesLegacyAndSignedContracts(t *testing.T) {
	const version = "0.7.0"
	dir := writeReleaseDir(t, version)
	checksumsPath := filepath.Join(dir, "checksums.txt")
	err := runGenerate(generateOptions{
		version:     version,
		releaseDate: "2026-08-12",
		dir:         dir,
		outputDir:   dir,
		checksums:   checksumsPath,
		getenv: func(string) string {
			return fixtureSeedsJSON(t, [][]byte{fixtureSeedA})
		},
		stdout: io.Discard,
	})
	if err != nil {
		t.Fatalf("runGenerate: %v", err)
	}

	// 旧契约（v0.6.0 更新器）：checksums.txt 每个归档恰好一条且哈希匹配。
	checksums, err := os.ReadFile(checksumsPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(dir, "release-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	signatureBytes, err := os.ReadFile(filepath.Join(dir, "release-manifest.sig"))
	if err != nil {
		t.Fatal(err)
	}
	for _, platform := range releasePlatforms {
		extension := ".tar.gz"
		if platform.goos == "windows" {
			extension = ".zip"
		}
		name := "javdb-cli_" + version + "_" + platform.goos + "_" + platform.goarch + extension
		archiveBytes, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("archive %q missing from fixture: %v", name, err)
		}
		sum := sha256.Sum256(archiveBytes)
		legacy := legacyChecksumLookup(t, checksums, name)
		if legacy != hex.EncodeToString(sum[:]) {
			t.Errorf("legacy contract hash mismatch for %s", name)
		}
	}

	// 新契约（v0.6.1/v0.7 更新器）：验签 + 双哈希。
	releaseManifest, err := manifest.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if len(releaseManifest.Targets) != 6 {
		t.Fatalf("manifest targets = %d, want six-platform set", len(releaseManifest.Targets))
	}
	signatures, err := manifest.ParseSignatures(signatureBytes)
	if err != nil {
		t.Fatalf("ParseSignatures: %v", err)
	}
	ring := manifest.NewKeyring()
	if err := ring.Add(ed25519PublicKey(t, fixtureSeedA)); err != nil {
		t.Fatal(err)
	}
	if err := ring.VerifySignatures(manifestBytes, signatures); err != nil {
		t.Fatalf("VerifySignatures: %v", err)
	}
	seen := map[string]bool{}
	for _, target := range releaseManifest.Targets {
		platform := target.GOOS + "/" + target.GOARCH
		if seen[platform] {
			t.Fatalf("duplicate target %s", platform)
		}
		seen[platform] = true
		archiveBytes, err := os.ReadFile(filepath.Join(dir, target.Archive))
		if err != nil {
			t.Fatalf("manifest archive %q missing: %v", target.Archive, err)
		}
		archiveSum := sha256.Sum256(archiveBytes)
		if target.ArchiveSHA256 != hex.EncodeToString(archiveSum[:]) {
			t.Errorf("archive hash mismatch for %s", target.Archive)
		}
		binary, err := archive.ExtractBinaryBytes(archiveBytes, target.Archive, target.Binary)
		if err != nil {
			t.Fatalf("extract %s: %v", target.Archive, err)
		}
		binarySum := sha256.Sum256(binary)
		if target.BinarySHA256 != hex.EncodeToString(binarySum[:]) {
			t.Errorf("binary hash mismatch for %s", target.Binary)
		}
	}
	if len(seen) != 6 {
		t.Fatalf("target set = %d, want 6", len(seen))
	}
}
