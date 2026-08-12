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
