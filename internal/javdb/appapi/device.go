package appapi

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func base64Decode(s string) ([]byte, error) {
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawStdEncoding.DecodeString(s)
}

func newDeviceUUID() string {
	if id, err := uuid.NewRandom(); err == nil {
		return id.String()
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// LoadOrCreateDeviceUUID returns a stable device uuid from path, creating if missing.
func LoadOrCreateDeviceUUID(path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		s := strings.TrimSpace(string(data))
		if s != "" {
			return s, nil
		}
	}
	id := newDeviceUUID()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return id, err
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return id, err
	}
	return id, nil
}
