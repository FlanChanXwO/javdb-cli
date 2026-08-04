package appapi

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeImagePayloadUnwrapsXORResponse(t *testing.T) {
	want := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10}
	const key = byte(0x97)
	raw := make([]byte, len(want)+1)
	raw[0] = key
	for i, b := range want {
		raw[i+1] = b ^ key
	}

	got, err := decodeImagePayload(raw)
	if err != nil {
		t.Fatalf("decode image payload: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("decoded image = %x, want %x", got, want)
	}
}

func TestFetchMediaRejectsNon2xxResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		_, _ = w.Write([]byte("gone"))
	}))
	defer server.Close()

	client, err := New(Options{})
	if err != nil {
		t.Fatalf("new app API client: %v", err)
	}
	_, err = client.fetchMedia(server.URL)
	if err == nil || !strings.Contains(err.Error(), "HTTP 410") {
		t.Fatalf("fetch media error = %v, want HTTP status error", err)
	}
}

func TestDownloadHLSDecryptsVODWithSequenceIV(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	key := []byte("0123456789abcdef")
	first := []byte("first HLS segment")
	second := []byte("second HLS segment")
	resources := map[string][]byte{
		playlistURL: []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-MEDIA-SEQUENCE:7\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:1.0,\nfirst.ts\n#EXTINF:1.0,\nsecond.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/previews/key.bin":   key,
		"https://media.example.test/previews/first.ts":  encryptHLSTestPayload(t, first, key, hlsSequenceIV(7)),
		"https://media.example.test/previews/second.ts": encryptHLSTestPayload(t, second, key, hlsSequenceIV(8)),
	}
	fetch := func(uri string) ([]byte, error) {
		body, ok := resources[uri]
		if !ok {
			return nil, fmt.Errorf("unexpected media URI %q", uri)
		}
		return body, nil
	}

	target := filepath.Join(t.TempDir(), "preview.ts")
	n, err := downloadHLS(fetch, playlistURL, target)
	if err != nil {
		t.Fatalf("download HLS: %v", err)
	}
	want := append(append([]byte(nil), first...), second...)
	if n != int64(len(want)) {
		t.Fatalf("written bytes = %d, want %d", n, len(want))
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read downloaded video: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("downloaded video = %q, want %q", got, want)
	}
}

func TestDownloadHLSRejectsUnfinishedPlaylistWithoutCreatingFile(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	target := filepath.Join(t.TempDir(), "preview.ts")
	_, err := downloadHLS(func(uri string) ([]byte, error) {
		return []byte("#EXTM3U\n#EXTINF:1.0,\nsegment.ts\n"), nil
	}, playlistURL, target)
	if err == nil {
		t.Fatal("unfinished HLS playlist unexpectedly succeeded")
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("unfinished playlist left output file: %v", statErr)
	}
}

func TestDownloadHLSRejectsInvalidPKCS7PaddingWithoutOutput(t *testing.T) {
	const playlistURL = "https://media.example.test/previews/index.m3u8"
	key := []byte("0123456789abcdef")
	invalidPlaintext := bytes.Repeat([]byte{0x01}, aes.BlockSize)
	invalidPlaintext[len(invalidPlaintext)-1] = 0x02
	resources := map[string][]byte{
		playlistURL: []byte("#EXTM3U\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:1.0,\nsegment.ts\n#EXT-X-ENDLIST\n"),
		"https://media.example.test/previews/key.bin":    key,
		"https://media.example.test/previews/segment.ts": encryptHLSRawPayload(t, invalidPlaintext, key, hlsSequenceIV(0)),
	}
	fetch := func(uri string) ([]byte, error) {
		body, ok := resources[uri]
		if !ok {
			return nil, fmt.Errorf("unexpected media URI %q", uri)
		}
		return body, nil
	}

	target := filepath.Join(t.TempDir(), "preview.ts")
	_, err := downloadHLS(fetch, playlistURL, target)
	if err == nil || !strings.Contains(err.Error(), "PKCS#7") {
		t.Fatalf("download HLS error = %v, want invalid padding", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("invalid padding left output file: %v", statErr)
	}
}

func TestWriteNewMediaFileNeverOverwritesExistingOutput(t *testing.T) {
	target := filepath.Join(t.TempDir(), "existing.jpg")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := writeNewMediaFile(target, func(w io.Writer) (int64, error) {
		n, writeErr := w.Write([]byte("replacement"))
		return int64(n), writeErr
	})
	if err == nil {
		t.Fatal("existing output unexpectedly overwritten")
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read fixture: %v", readErr)
	}
	if string(got) != "original" {
		t.Fatalf("existing output changed to %q", got)
	}
}

func encryptHLSTestPayload(t *testing.T, plaintext, key, iv []byte) []byte {
	t.Helper()
	padding := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := append([]byte(nil), plaintext...)
	for i := 0; i < padding; i++ {
		padded = append(padded, byte(padding))
	}
	return encryptHLSRawPayload(t, padded, key, iv)
}

func encryptHLSRawPayload(t *testing.T, plaintext, key, iv []byte) []byte {
	t.Helper()
	if len(plaintext)%aes.BlockSize != 0 {
		t.Fatalf("plaintext length %d is not a cipher block multiple", len(plaintext))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("new AES cipher: %v", err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	return ciphertext
}
