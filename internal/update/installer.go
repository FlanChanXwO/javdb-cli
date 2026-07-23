package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const checksumsAssetName = "checksums.txt"

// BinaryChecker validates a downloaded binary before it can replace javdb.
type BinaryChecker interface {
	Check(context.Context, string, string) error
}

// BinaryCheckerFunc adapts a function for installer tests.
type BinaryCheckerFunc func(context.Context, string, string) error

// Check invokes the wrapped binary checker.
func (f BinaryCheckerFunc) Check(ctx context.Context, path, version string) error {
	return f(ctx, path, version)
}

// ReleaseInstallerOptions injects system boundaries for deterministic tests.
type ReleaseInstallerOptions struct {
	HTTPClient        *http.Client
	ExecutablePath    func() (string, error)
	GOOS              string
	GOARCH            string
	BinaryChecker     BinaryChecker
	Replacer          func(string, string) error
	AssetURLValidator func(Release, ReleaseAsset) error
}

type releaseInstaller struct {
	httpClient        *http.Client
	executablePath    func() (string, error)
	goos              string
	goarch            string
	binaryChecker     BinaryChecker
	replacer          func(string, string) error
	assetURLValidator func(Release, ReleaseAsset) error
}

// NewReleaseInstaller creates the production archive updater. Every write is
// delayed until the archive checksum and the embedded binary version both match.
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
	binaryChecker := options.BinaryChecker
	if binaryChecker == nil {
		binaryChecker = processBinaryChecker{}
	}
	replacer := options.Replacer
	if replacer == nil {
		replacer = replaceExecutable
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
		binaryChecker:     binaryChecker,
		replacer:          replacer,
		assetURLValidator: assetURLValidator,
	}
}

// Install obtains and verifies exactly one archive for the running platform.
// On any failed validation the currently installed executable remains unchanged.
func (i *releaseInstaller) Install(ctx context.Context, release Release) (resultErr error) {
	if i == nil {
		return fmt.Errorf("release installer is nil")
	}
	if _, err := parseSemanticVersion(release.TagName); err != nil {
		return fmt.Errorf("parse release tag %q: %w", release.TagName, err)
	}
	assets, err := selectReleaseAssets(release, releaseArchiveName(release.TagName, i.goos, i.goarch))
	if err != nil {
		return err
	}
	for _, asset := range []ReleaseAsset{assets.archive, assets.checksums} {
		if err := i.assetURLValidator(release, asset); err != nil {
			return err
		}
	}
	checksums, err := i.download(ctx, assets.checksums)
	if err != nil {
		return fmt.Errorf("download %s: %w", checksumsAssetName, err)
	}
	expectedChecksum, err := checksumForArchive(checksums, assets.archive.Name)
	if err != nil {
		return err
	}
	archive, err := i.download(ctx, assets.archive)
	if err != nil {
		return fmt.Errorf("download release archive %q: %w", assets.archive.Name, err)
	}
	actualChecksum := sha256.Sum256(archive)
	if hex.EncodeToString(actualChecksum[:]) != expectedChecksum {
		return fmt.Errorf("release archive %q SHA-256 does not match checksums.txt", assets.archive.Name)
	}
	target, err := i.executablePath()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}
	target, err = resolveExecutablePath(target)
	if err != nil {
		return err
	}
	workDirectory, err := os.MkdirTemp(filepath.Dir(target), ".javdb-update-")
	if err != nil {
		return fmt.Errorf("create update temporary directory beside %q: %w", target, err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(workDirectory); cleanupErr != nil && resultErr == nil {
			resultErr = fmt.Errorf("remove update temporary directory %q: %w", workDirectory, cleanupErr)
		}
	}()
	candidate := filepath.Join(workDirectory, executableName(i.goos))
	if err := extractReleaseBinary(archive, assets.archive.Name, candidate, executableName(i.goos)); err != nil {
		return err
	}
	if err := i.binaryChecker.Check(ctx, candidate, release.TagName); err != nil {
		return fmt.Errorf("preflight downloaded executable %q: %w", candidate, err)
	}
	staging, err := os.CreateTemp(filepath.Dir(target), ".javdb-update-stage-")
	if err != nil {
		return fmt.Errorf("create staged update beside %q: %w", target, err)
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
	if err := os.Rename(candidate, stagingPath); err != nil {
		return fmt.Errorf("stage verified update %q: %w", stagingPath, err)
	}
	if err := i.replacer(stagingPath, target); err != nil {
		return fmt.Errorf("replace executable %q: %w", target, err)
	}
	return nil
}

type selectedReleaseAssets struct {
	archive   ReleaseAsset
	checksums ReleaseAsset
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
		case checksumsAssetName:
			if selected.checksums.Name != "" {
				return selectedReleaseAssets{}, fmt.Errorf("release contains duplicate asset %q", checksumsAssetName)
			}
			selected.checksums = asset
		}
	}
	if selected.archive.Name == "" {
		return selectedReleaseAssets{}, fmt.Errorf("release has no platform archive asset %q", archiveName)
	}
	if selected.checksums.Name == "" {
		return selectedReleaseAssets{}, fmt.Errorf("release has no asset %q", checksumsAssetName)
	}
	if selected.archive.DownloadURL == "" || selected.checksums.DownloadURL == "" {
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
	expected := "https://github.com/" + githubRepository + "/releases/download/" + release.TagName + "/" + asset.Name
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

func checksumForArchive(checksums []byte, archiveName string) (string, error) {
	var expected string
	for _, line := range strings.Split(string(checksums), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(fields[1], "*") != archiveName {
			continue
		}
		if expected != "" {
			return "", fmt.Errorf("checksums.txt has duplicate entry for %q", archiveName)
		}
		if len(fields[0]) != sha256.Size*2 || strings.ToLower(fields[0]) != fields[0] {
			return "", fmt.Errorf("checksums.txt has invalid SHA-256 for %q", archiveName)
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return "", fmt.Errorf("checksums.txt has invalid SHA-256 for %q: %w", archiveName, err)
		}
		expected = fields[0]
	}
	if expected == "" {
		return "", fmt.Errorf("checksums.txt has no entry for %q", archiveName)
	}
	return expected, nil
}

func resolveExecutablePath(executablePath string) (string, error) {
	target, err := filepath.Abs(executablePath)
	if err != nil {
		return "", fmt.Errorf("resolve current executable %q: %w", executablePath, err)
	}
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return target, nil
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlink %q: %w", target, err)
	}
	return filepath.Abs(resolved)
}

func extractReleaseBinary(archive []byte, archiveName, destination, binaryName string) error {
	if strings.HasSuffix(archiveName, ".tar.gz") {
		return extractTarGzBinary(archive, destination, binaryName)
	}
	if strings.HasSuffix(archiveName, ".zip") {
		return extractZIPBinary(archive, destination, binaryName)
	}
	return fmt.Errorf("unsupported release archive %q", archiveName)
}

func extractTarGzBinary(archive []byte, destination, binaryName string) error {
	gzipped, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return fmt.Errorf("open tar.gz release archive: %w", err)
	}
	defer gzipped.Close()
	reader := tar.NewReader(gzipped)
	found := false
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar.gz release archive: %w", err)
		}
		if header.Name != binaryName {
			continue
		}
		if found || !header.FileInfo().Mode().IsRegular() {
			return fmt.Errorf("release archive has an invalid binary entry %q", binaryName)
		}
		if err := writeExtractedBinary(destination, reader); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return fmt.Errorf("release archive has no binary entry %q", binaryName)
	}
	return nil
}

func extractZIPBinary(archive []byte, destination, binaryName string) error {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return fmt.Errorf("open zip release archive: %w", err)
	}
	var binary *zip.File
	for _, file := range reader.File {
		if file.Name != binaryName {
			continue
		}
		if binary != nil || !file.FileInfo().Mode().IsRegular() {
			return fmt.Errorf("release archive has an invalid binary entry %q", binaryName)
		}
		binary = file
	}
	if binary == nil {
		return fmt.Errorf("release archive has no binary entry %q", binaryName)
	}
	entry, err := binary.Open()
	if err != nil {
		return fmt.Errorf("open zip binary entry %q: %w", binaryName, err)
	}
	defer entry.Close()
	return writeExtractedBinary(destination, entry)
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

type processBinaryChecker struct{}

func (processBinaryChecker) Check(ctx context.Context, executablePath, expectedVersion string) error {
	command := exec.CommandContext(ctx, executablePath, "version", "--json")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("run candidate version command: %w", err)
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(output, &version); err != nil {
		return fmt.Errorf("decode candidate version JSON: %w", err)
	}
	if version.Version != expectedVersion {
		return fmt.Errorf("candidate reports version %q, expected %q", version.Version, expectedVersion)
	}
	return nil
}
