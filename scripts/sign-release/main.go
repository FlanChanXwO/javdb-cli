// Command sign-release 从受保护环境私钥生成并签名 v1 发布清单。
//
// 私钥只从 JAVDB_RELEASE_ED25519_PRIVATE_KEYS 环境变量读取（JSON 数组，
// 每项是一个标准 Base64 编码的 32-byte Ed25519 seed）；工具不把私钥写入
// 磁盘、artifact 或日志，错误只报告索引与非敏感原因。
package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/update/archive"
	"github.com/FlanChanXwO/javdb-cli/internal/update/manifest"
)

// privateKeysEnvironment 是 GitHub release environment secret 的固定名称。
const privateKeysEnvironment = "JAVDB_RELEASE_ED25519_PRIVATE_KEYS"

// releasePlatforms 是发布契约固定的六平台顺序（与 release workflow 一致）。
var releasePlatforms = []struct{ goos, goarch string }{
	{goos: "darwin", goarch: "amd64"},
	{goos: "darwin", goarch: "arm64"},
	{goos: "linux", goarch: "amd64"},
	{goos: "linux", goarch: "arm64"},
	{goos: "windows", goarch: "amd64"},
	{goos: "windows", goarch: "arm64"},
}

type generateOptions struct {
	version     string
	releaseDate string
	dir         string
	outputDir   string
	checksums   string
	getenv      func(string) string
	stdout      io.Writer
}

func main() {
	flags := flag.NewFlagSet("sign-release", flag.ExitOnError)
	version := flags.String("version", "", "release version without the v prefix, e.g. 0.7.0")
	releaseDate := flags.String("release-date", "", "release date in YYYY-MM-DD, matching the audited changelog")
	dir := flags.String("dir", ".", "directory containing the six verified platform archives")
	output := flags.String("output", "", "output directory for release-manifest.json and release-manifest.sig (default: --dir)")
	checksums := flags.String("checksums", "", "output path for checksums.txt derived from the manifest (optional)")
	showKeys := flags.Bool("show-keys", false, "print the derived public key and key_id for each environment seed, then exit")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "usage: sign-release --version VERSION --release-date DATE [--dir DIR] [--output DIR] [--checksums PATH]\n       sign-release --show-keys\n")
		fmt.Fprintf(flags.Output(), "\nReads private key seeds from the %s environment variable and signs a\nv1 release manifest over the six platform archives in --dir.\n", privateKeysEnvironment)
		flags.PrintDefaults()
	}
	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *showKeys {
		if err := runShowKeys(os.Getenv(privateKeysEnvironment), os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "sign-release: %v\n", err)
			os.Exit(1)
		}
		return
	}
	outputDir := *output
	if outputDir == "" {
		outputDir = *dir
	}
	err := runGenerate(generateOptions{
		version:     *version,
		releaseDate: *releaseDate,
		dir:         *dir,
		outputDir:   outputDir,
		checksums:   *checksums,
		getenv:      os.Getenv,
		stdout:      os.Stdout,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign-release: %v\n", err)
		os.Exit(1)
	}
}

// runShowKeys 打印每个环境 seed 派生的公钥与 key_id，用于把生产公钥登记到
// DefaultKeyring；绝不打印 seed、私钥或原始环境值。
func runShowKeys(environmentValue string, stdout io.Writer) error {
	seeds, err := seedsFromEnvironment(environmentValue)
	if err != nil {
		return err
	}
	for index, seed := range seeds {
		publicKey := ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey)
		fmt.Fprintf(stdout, "key %d: key_id=%s public_key_hex=%s\n", index, manifest.KeyID(publicKey), hex.EncodeToString(publicKey))
	}
	return nil
}

// runGenerate 计算六平台归档与内部二进制的 SHA-256、生成规范清单并用环境
// 中全部私钥签名，最后写入 release-manifest.json 与 release-manifest.sig。
func runGenerate(options generateOptions) error {
	if options.version == "" {
		return fmt.Errorf("--version is required")
	}
	if options.releaseDate == "" {
		return fmt.Errorf("--release-date is required")
	}
	version := strings.TrimPrefix(options.version, "v")
	tag := "v" + version

	seeds, err := seedsFromEnvironment(options.getenv(privateKeysEnvironment))
	if err != nil {
		return err
	}

	entries, err := collectPlatformArchives(options.dir, version)
	if err != nil {
		return err
	}
	targets := make([]manifest.Target, 0, len(entries))
	for _, entry := range entries {
		content, err := os.ReadFile(entry.path)
		if err != nil {
			return fmt.Errorf("read release archive %q: %w", entry.name, err)
		}
		archiveSum := sha256.Sum256(content)
		binaryContent, err := archive.ExtractBinaryBytes(content, entry.name, entry.binary)
		if err != nil {
			return err
		}
		binarySum := sha256.Sum256(binaryContent)
		targets = append(targets, manifest.Target{
			GOOS:          entry.goos,
			GOARCH:        entry.goarch,
			Archive:       entry.name,
			ArchiveSHA256: hex.EncodeToString(archiveSum[:]),
			Binary:        entry.binary,
			BinarySHA256:  hex.EncodeToString(binarySum[:]),
		})
	}

	releaseManifest, err := manifest.NewManifest(tag, options.releaseDate, targets)
	if err != nil {
		return err
	}
	manifestBytes, err := releaseManifest.Canonical()
	if err != nil {
		return fmt.Errorf("encode release manifest: %w", err)
	}
	signatures, err := manifest.SignManifest(manifestBytes, seeds)
	if err != nil {
		return err
	}
	signatureBytes, err := signatures.Canonical()
	if err != nil {
		return fmt.Errorf("encode release signatures: %w", err)
	}

	manifestPath := filepath.Join(options.outputDir, "release-manifest.json")
	if err := writeFileExclusive(manifestPath, manifestBytes); err != nil {
		return err
	}
	signaturePath := filepath.Join(options.outputDir, "release-manifest.sig")
	if err := writeFileExclusive(signaturePath, signatureBytes); err != nil {
		return err
	}
	if options.checksums != "" {
		// checksums.txt 由清单派生（archive 哈希），保持 v0.6.x 更新器、
		// Homebrew 与人工校验兼容的 `<hex>  <archive>` 格式。
		if err := writeFileExclusive(options.checksums, deriveChecksums(releaseManifest)); err != nil {
			return err
		}
	}
	if options.stdout != nil {
		fmt.Fprintf(options.stdout, "signed release manifest %s with %d key(s)\n", tag, len(signatures.Signatures))
		fmt.Fprintf(options.stdout, "wrote %s\nwrote %s\n", manifestPath, signaturePath)
		if options.checksums != "" {
			fmt.Fprintf(options.stdout, "wrote %s\n", options.checksums)
		}
	}
	return nil
}

// deriveChecksums 从清单的归档哈希派生兼容 checksums.txt 内容。
func deriveChecksums(releaseManifest *manifest.Manifest) []byte {
	var builder strings.Builder
	for _, target := range releaseManifest.Targets {
		fmt.Fprintf(&builder, "%s  %s\n", target.ArchiveSHA256, target.Archive)
	}
	return []byte(builder.String())
}

type platformArchive struct {
	goos, goarch string
	name         string
	binary       string
	path         string
}

// collectPlatformArchives 定位六平台归档；缺少任何平台都显式失败。
func collectPlatformArchives(dir, version string) ([]platformArchive, error) {
	entries := make([]platformArchive, 0, len(releasePlatforms))
	for _, platform := range releasePlatforms {
		extension := ".tar.gz"
		binary := "javdb"
		if platform.goos == "windows" {
			extension = ".zip"
			binary = "javdb.exe"
		}
		name := "javdb-cli_" + version + "_" + platform.goos + "_" + platform.goarch + extension
		path := filepath.Join(dir, name)
		info, err := os.Lstat(path)
		if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("release archive is missing or not a regular file: %s", name)
		}
		entries = append(entries, platformArchive{
			goos:   platform.goos,
			goarch: platform.goarch,
			name:   name,
			binary: binary,
			path:   path,
		})
	}
	return entries, nil
}

// seedsFromEnvironment 解析环境变量中的 Base64 seed 数组；错误只报告
// 数组项索引与非敏感原因，绝不包含 seed、私钥或展开后的值。
func seedsFromEnvironment(value string) ([][]byte, error) {
	if value == "" {
		return nil, fmt.Errorf("environment variable %s is not set", privateKeysEnvironment)
	}
	var encoded []string
	if err := json.Unmarshal([]byte(value), &encoded); err != nil {
		return nil, fmt.Errorf("environment variable %s must be a JSON array of Base64 seeds: %w", privateKeysEnvironment, err)
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("environment variable %s must contain at least one seed", privateKeysEnvironment)
	}
	seeds := make([][]byte, 0, len(encoded))
	for index, item := range encoded {
		seed, err := base64.StdEncoding.DecodeString(item)
		if err != nil {
			return nil, fmt.Errorf("private key seed %d is not valid standard Base64", index)
		}
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("private key seed %d must decode to %d bytes, got %d", index, ed25519.SeedSize, len(seed))
		}
		seeds = append(seeds, seed)
	}
	return seeds, nil
}

// writeFileExclusive 以独占模式写文件，已存在的输出显式失败。
func writeFileExclusive(path string, content []byte) error {
	output, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create output %q: %w", path, err)
	}
	if _, err := output.Write(content); err != nil {
		_ = output.Close()
		return fmt.Errorf("write output %q: %w", path, err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close output %q: %w", path, err)
	}
	return nil
}
