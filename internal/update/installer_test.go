package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseInstallerVerifiesAndReplacesTarBinary(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("verified linux binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	checker := BinaryCheckerFunc(func(_ context.Context, path, version string) error {
		if version != "v0.2.0" {
			t.Fatalf("version = %q", version)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(body) != "verified linux binary" {
			t.Fatalf("candidate = %q", body)
		}
		return nil
	})
	installAndAssert(t, archive, "javdb-cli_0.2.0_linux_amd64.tar.gz", "linux", "amd64", target, "verified linux binary", checker)
}

func TestReleaseInstallerVerifiesAndReplacesZIPBinary(t *testing.T) {
	archive := zipArchive(t, "javdb.exe", []byte("verified windows binary"))
	target := filepath.Join(t.TempDir(), "javdb.exe")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	checker := BinaryCheckerFunc(func(_ context.Context, path, version string) error {
		if version != "v0.2.0" {
			t.Fatalf("version = %q", version)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(body) != "verified windows binary" {
			t.Fatalf("candidate = %q", body)
		}
		return nil
	})
	installAndAssert(t, archive, "javdb-cli_0.2.0_windows_arm64.zip", "windows", "arm64", target, "verified windows binary", checker)
}

func TestReleaseInstallerKeepsCurrentBinaryOnChecksumMismatch(t *testing.T) {
	archive := tarGzArchive(t, "javdb", []byte("untrusted binary"))
	target := filepath.Join(t.TempDir(), "javdb")
	if err := os.WriteFile(target, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/checksums.txt":
			_, _ = io.WriteString(writer, "0000000000000000000000000000000000000000000000000000000000000000  javdb-cli_0.2.0_linux_amd64.tar.gz\n")
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
		BinaryChecker: BinaryCheckerFunc(func(context.Context, string, string) error {
			t.Fatal("binary checker ran before checksum verification")
			return nil
		}),
		Replacer: os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	release := Release{TagName: "v0.2.0", Assets: []ReleaseAsset{
		{Name: "checksums.txt", DownloadURL: server.URL + "/checksums.txt"},
		{Name: "javdb-cli_0.2.0_linux_amd64.tar.gz", DownloadURL: server.URL + "/archive"},
	}}
	if err := installer.Install(context.Background(), release); err == nil {
		t.Fatal("checksum mismatch unexpectedly installed archive")
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "old binary" {
		t.Fatalf("target changed to %q", body)
	}
}

func installAndAssert(t *testing.T, archive []byte, archiveName, goos, goarch, target, want string, checker BinaryChecker) {
	t.Helper()
	sum := sha256.Sum256(archive)
	checksums := hex.EncodeToString(sum[:]) + "  " + archiveName + "\n"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/checksums.txt":
			_, _ = io.WriteString(writer, checksums)
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
		BinaryChecker:  checker,
		Replacer:       os.Rename,
		AssetURLValidator: func(Release, ReleaseAsset) error {
			return nil
		},
	})
	release := Release{TagName: "v0.2.0", Assets: []ReleaseAsset{
		{Name: "checksums.txt", DownloadURL: server.URL + "/checksums.txt"},
		{Name: archiveName, DownloadURL: server.URL + "/archive"},
	}}
	if err := installer.Install(context.Background(), release); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Fatalf("installed binary = %q, want %q", body, want)
	}
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
