// Package archive verifies and installs platform release archives.
package archive

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/update/manifest"
	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
	"github.com/FlanChanXwO/javdb-cli/internal/update/process"
	"github.com/FlanChanXwO/javdb-cli/internal/update/release"
)

const (
	manifestAssetName   = "release-manifest.json"
	signaturesAssetName = "release-manifest.sig"
)

// Release 与 ReleaseAsset 保持 archive 包内部使用同一 wire model。
type Release = model.Release
type ReleaseAsset = model.ReleaseAsset

// ReleaseInstaller installs an already verified release for the current platform.
type ReleaseInstaller interface {
	Install(context.Context, model.Release) error
}

// ReleaseInstallerOptions injects system boundaries for deterministic tests.
type ReleaseInstallerOptions struct {
	HTTPClient        *http.Client
	ExecutablePath    func() (string, error)
	GOOS              string
	GOARCH            string
	Keyring           *manifest.Keyring
	Replacer          func(string, string) error
	AssetURLValidator func(model.Release, model.ReleaseAsset) error
}

type releaseInstaller struct {
	httpClient        *http.Client
	executablePath    func() (string, error)
	goos              string
	goarch            string
	keyring           *manifest.Keyring
	replacer          func(string, string) error
	assetURLValidator func(Release, ReleaseAsset) error
}

// NewReleaseInstaller creates the production archive updater. Every write is
// delayed until the signed release manifest, its Ed25519 signatures, the
// archive SHA-256 and the extracted binary SHA-256 all match.
func NewReleaseInstaller(options ReleaseInstallerOptions) ReleaseInstaller {
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	executablePath := options.ExecutablePath
	if executablePath == nil {
		executablePath = os.Executable
	}
	goos := options.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := options.GOARCH
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	keyring := options.Keyring
	if keyring == nil {
		keyring = manifest.DefaultKeyring()
	}
	replacer := options.Replacer
	if replacer == nil {
		replacer = process.ReplaceExecutable
	}
	assetURLValidator := options.AssetURLValidator
	if assetURLValidator == nil {
		assetURLValidator = validateOfficialReleaseAssetURL
	}
	return &releaseInstaller{
		httpClient:        httpClient,
		executablePath:    executablePath,
		goos:              goos,
		goarch:            goarch,
		keyring:           keyring,
		replacer:          replacer,
		assetURLValidator: assetURLValidator,
	}
}

// Install obtains and verifies exactly one archive for the running platform.
// Verification order is fixed: official asset URLs, manifest signatures over
// the raw manifest bytes, strict v1 manifest binding to this release and
// platform, archive SHA-256, then binary SHA-256. The candidate binary is
// never executed. On any failed validation the currently installed
// executable remains unchanged.
func (i *releaseInstaller) Install(ctx context.Context, candidateRelease Release) (resultErr error) {
	if i == nil {
		return fmt.Errorf("release installer is nil")
	}
	if _, err := release.ParseSemanticVersion(candidateRelease.TagName); err != nil {
		return fmt.Errorf("parse release tag %q: %w", candidateRelease.TagName, err)
	}
	assets, err := selectReleaseAssets(candidateRelease, releaseArchiveName(candidateRelease.TagName, i.goos, i.goarch))
	if err != nil {
		return err
	}
	for _, asset := range []ReleaseAsset{assets.archive, assets.manifest, assets.signatures} {
		if err := i.assetURLValidator(candidateRelease, asset); err != nil {
			return err
		}
	}
	manifestBytes, err := i.download(ctx, assets.manifest)
	if err != nil {
		return fmt.Errorf("download %s: %w", manifestAssetName, err)
	}
	signatureBytes, err := i.download(ctx, assets.signatures)
	if err != nil {
		return fmt.Errorf("download %s: %w", signaturesAssetName, err)
	}
	signatures, err := manifest.ParseSignatures(signatureBytes)
	if err != nil {
		return err
	}
	if err := i.keyring.VerifySignatures(manifestBytes, signatures); err != nil {
		return fmt.Errorf("verify release manifest signature: %w", err)
	}
	releaseManifest, err := manifest.ParseManifest(manifestBytes)
	if err != nil {
		return err
	}
	target, err := selectManifestTarget(releaseManifest, candidateRelease.TagName, i.goos, i.goarch)
	if err != nil {
		return err
	}
	if target.Archive != assets.archive.Name {
		return fmt.Errorf("release manifest selects archive %q but release assets contain %q", target.Archive, assets.archive.Name)
	}
	archive, err := i.download(ctx, assets.archive)
	if err != nil {
		return fmt.Errorf("download release archive %q: %w", assets.archive.Name, err)
	}
	if err := matchSHA256(archive, target.ArchiveSHA256, "release archive "+assets.archive.Name); err != nil {
		return err
	}
	binaryContent, err := ExtractBinaryBytes(archive, assets.archive.Name, target.Binary)
	if err != nil {
		return err
	}
	if err := matchSHA256(binaryContent, target.BinarySHA256, "release binary "+target.Binary); err != nil {
		return err
	}
	executable, err := i.executablePath()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}
	executable, err = process.ResolveExecutablePath(executable)
	if err != nil {
		return err
	}
	staging, err := os.CreateTemp(filepath.Dir(executable), ".javdb-update-stage-")
	if err != nil {
		return fmt.Errorf("create staged update beside %q: %w", executable, err)
	}
	stagingPath := staging.Name()
	if err := staging.Close(); err != nil {
		return fmt.Errorf("close staged update %q: %w", stagingPath, err)
	}
	defer func() {
		if err := os.Remove(stagingPath); err != nil && !os.IsNotExist(err) && resultErr == nil {
			resultErr = fmt.Errorf("remove staged update %q: %w", stagingPath, err)
		}
	}()
	if err := os.Remove(stagingPath); err != nil {
		return fmt.Errorf("prepare staged update %q: %w", stagingPath, err)
	}
	if err := writeExtractedBinary(stagingPath, bytes.NewReader(binaryContent)); err != nil {
		return err
	}
	if err := i.replacer(stagingPath, executable); err != nil {
		return fmt.Errorf("replace executable %q: %w", executable, err)
	}
	return nil
}

// selectManifestTarget binds the signed manifest to the selected release tag
// and the running platform. The manifest repository and stable SemVer tag are
// already enforced by ParseManifest.
func selectManifestTarget(releaseManifest *manifest.Manifest, tag, goos, goarch string) (*manifest.Target, error) {
	if releaseManifest.Tag != tag {
		return nil, fmt.Errorf("release manifest tag %q does not match release tag %q", releaseManifest.Tag, tag)
	}
	var selected *manifest.Target
	for index := range releaseManifest.Targets {
		target := &releaseManifest.Targets[index]
		if target.GOOS != goos || target.GOARCH != goarch {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("release manifest has duplicate target %s/%s", goos, goarch)
		}
		selected = target
	}
	if selected == nil {
		return nil, fmt.Errorf("release manifest has no target for platform %s/%s", goos, goarch)
	}
	return selected, nil
}

// matchSHA256 校验 content 与清单中声明的十六进制 SHA-256 一致。
func matchSHA256(content []byte, expectedHex, description string) error {
	sum := sha256.Sum256(content)
	actualHex := hex.EncodeToString(sum[:])
	if actualHex != expectedHex {
		return fmt.Errorf("%s SHA-256 does not match the release manifest", description)
	}
	return nil
}

type selectedReleaseAssets struct {
	archive    ReleaseAsset
	manifest   ReleaseAsset
	signatures ReleaseAsset
}

func selectReleaseAssets(release Release, archiveName string) (selectedReleaseAssets, error) {
	var selected selectedReleaseAssets
	for _, asset := range release.Assets {
		switch asset.Name {
		case archiveName:
			if selected.archive.Name != "" {
				return selectedReleaseAssets{}, fmt.Errorf("release contains duplicate archive asset %q", archiveName)
			}
			selected.archive = asset
		case manifestAssetName:
			if selected.manifest.Name != "" {
				return selectedReleaseAssets{}, fmt.Errorf("release contains duplicate asset %q", manifestAssetName)
			}
			selected.manifest = asset
		case signaturesAssetName:
			if selected.signatures.Name != "" {
				return selectedReleaseAssets{}, fmt.Errorf("release contains duplicate asset %q", signaturesAssetName)
			}
			selected.signatures = asset
		}
	}
	if selected.archive.Name == "" {
		return selectedReleaseAssets{}, fmt.Errorf("release has no platform archive asset %q", archiveName)
	}
	if selected.manifest.Name == "" {
		return selectedReleaseAssets{}, fmt.Errorf("release has no asset %q", manifestAssetName)
	}
	if selected.signatures.Name == "" {
		return selectedReleaseAssets{}, fmt.Errorf("release has no asset %q", signaturesAssetName)
	}
	if selected.archive.DownloadURL == "" || selected.manifest.DownloadURL == "" || selected.signatures.DownloadURL == "" {
		return selectedReleaseAssets{}, fmt.Errorf("release asset download URL is empty")
	}
	return selected, nil
}

func releaseArchiveName(tag, goos, goarch string) string {
	extension := ".tar.gz"
	if goos == "windows" {
		extension = ".zip"
	}
	return "javdb-cli_" + strings.TrimPrefix(tag, "v") + "_" + goos + "_" + goarch + extension
}

func validateOfficialReleaseAssetURL(release Release, asset ReleaseAsset) error {
	expected := "https://github.com/" + model.GitHubRepository + "/releases/download/" + release.TagName + "/" + asset.Name
	if asset.DownloadURL != expected {
		return fmt.Errorf("release asset %q has untrusted download URL %q", asset.Name, asset.DownloadURL)
	}
	return nil
}

func (i *releaseInstaller) download(ctx context.Context, asset ReleaseAsset) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request for asset %q: %w", asset.Name, err)
	}
	response, err := i.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request asset %q: %w", asset.Name, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("asset %q returned HTTP %s", asset.Name, response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read asset %q: %w", asset.Name, err)
	}
	return body, nil
}

// ExtractBinaryBytes 从发布归档中提取唯一预期二进制并返回其字节。
// 归档内存在重复、非普通文件或缺失该二进制都会显式报错。
func ExtractBinaryBytes(archive []byte, archiveName, binaryName string) ([]byte, error) {
	if strings.HasSuffix(archiveName, ".tar.gz") {
		return extractTarGzBinaryBytes(archive, binaryName)
	}
	if strings.HasSuffix(archiveName, ".zip") {
		return extractZIPBinaryBytes(archive, binaryName)
	}
	return nil, fmt.Errorf("unsupported release archive %q", archiveName)
}

func extractReleaseBinary(archive []byte, archiveName, destination, binaryName string) error {
	content, err := ExtractBinaryBytes(archive, archiveName, binaryName)
	if err != nil {
		return err
	}
	return writeExtractedBinary(destination, bytes.NewReader(content))
}

func extractTarGzBinaryBytes(archive []byte, binaryName string) ([]byte, error) {
	gzipped, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("open tar.gz release archive: %w", err)
	}
	defer gzipped.Close()
	reader := tar.NewReader(gzipped)
	var content []byte
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar.gz release archive: %w", err)
		}
		if header.Name != binaryName {
			continue
		}
		if content != nil || !header.FileInfo().Mode().IsRegular() {
			return nil, fmt.Errorf("release archive has an invalid binary entry %q", binaryName)
		}
		content, err = io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("read binary entry %q from tar.gz release archive: %w", binaryName, err)
		}
	}
	if content == nil {
		return nil, fmt.Errorf("release archive has no binary entry %q", binaryName)
	}
	return content, nil
}

func extractZIPBinaryBytes(archive []byte, binaryName string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open zip release archive: %w", err)
	}
	var binary *zip.File
	for _, file := range reader.File {
		if file.Name != binaryName {
			continue
		}
		if binary != nil || !file.FileInfo().Mode().IsRegular() {
			return nil, fmt.Errorf("release archive has an invalid binary entry %q", binaryName)
		}
		binary = file
	}
	if binary == nil {
		return nil, fmt.Errorf("release archive has no binary entry %q", binaryName)
	}
	entry, err := binary.Open()
	if err != nil {
		return nil, fmt.Errorf("open zip binary entry %q: %w", binaryName, err)
	}
	defer entry.Close()
	content, err := io.ReadAll(entry)
	if err != nil {
		return nil, fmt.Errorf("read zip binary entry %q: %w", binaryName, err)
	}
	return content, nil
}

func writeExtractedBinary(destination string, source io.Reader) error {
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return fmt.Errorf("create extracted binary %q: %w", destination, err)
	}
	if _, err := io.Copy(output, source); err != nil {
		_ = output.Close()
		return fmt.Errorf("extract binary %q: %w", destination, err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close extracted binary %q: %w", destination, err)
	}
	return nil
}
