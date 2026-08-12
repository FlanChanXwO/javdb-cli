package archive

import (
	"testing"
)

func TestExtractBinaryBytesReturnsExpectedEntry(t *testing.T) {
	content := []byte("verified linux binary")
	extracted, err := ExtractBinaryBytes(tarGzArchive(t, "javdb", content), "javdb-cli_0.2.0_linux_amd64.tar.gz", "javdb")
	if err != nil {
		t.Fatalf("ExtractBinaryBytes tar.gz: %v", err)
	}
	if string(extracted) != string(content) {
		t.Errorf("tar.gz content = %q, want %q", extracted, content)
	}

	content = []byte("verified windows binary")
	extracted, err = ExtractBinaryBytes(zipArchive(t, "javdb.exe", content), "javdb-cli_0.2.0_windows_amd64.zip", "javdb.exe")
	if err != nil {
		t.Fatalf("ExtractBinaryBytes zip: %v", err)
	}
	if string(extracted) != string(content) {
		t.Errorf("zip content = %q, want %q", extracted, content)
	}
}

func TestExtractBinaryBytesRejectsMissingEntry(t *testing.T) {
	if _, err := ExtractBinaryBytes(tarGzArchive(t, "javdb", []byte("x")), "javdb-cli_0.2.0_linux_amd64.tar.gz", "javdb.exe"); err == nil {
		t.Error("ExtractBinaryBytes accepted archive without the expected binary")
	}
	if _, err := ExtractBinaryBytes(zipArchive(t, "javdb.exe", []byte("x")), "javdb-cli_0.2.0_windows_amd64.zip", "javdb"); err == nil {
		t.Error("ExtractBinaryBytes accepted zip without the expected binary")
	}
}

func TestExtractBinaryBytesRejectsUnsupportedArchive(t *testing.T) {
	if _, err := ExtractBinaryBytes([]byte("x"), "javdb-cli_0.2.0_linux_amd64.rar", "javdb"); err == nil {
		t.Error("ExtractBinaryBytes accepted unsupported archive format")
	}
}
