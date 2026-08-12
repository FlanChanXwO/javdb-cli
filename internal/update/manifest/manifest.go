// Package manifest 定义 v1 签名发布清单协议：发布清单、签名文件、规范 JSON
// 编码、Ed25519 签名与验签，以及客户端内置的公钥环。
package manifest

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/update/model"
)

const (
	// ManifestSchema 是发布清单文档的固定 schema。
	ManifestSchema = "javdb.release-manifest/v1"
	// SignaturesSchema 是签名文件的固定 schema。
	SignaturesSchema = "javdb.release-signatures/v1"
)

// DefaultRepository 是官方发布清单绑定的仓库；客户端只接受该值。
const DefaultRepository = model.GitHubRepository

// Target 描述一个平台归档及其内部二进制。
// 字段顺序即规范 JSON 编码顺序，禁止重排。
type Target struct {
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	Archive       string `json:"archive"`
	ArchiveSHA256 string `json:"archive_sha256"`
	Binary        string `json:"binary"`
	BinarySHA256  string `json:"binary_sha256"`
}

// Manifest 是 v1 签名发布清单。
// 字段顺序即规范 JSON 编码顺序，禁止重排。
type Manifest struct {
	Schema      string   `json:"schema"`
	Repository  string   `json:"repository"`
	Tag         string   `json:"tag"`
	Version     string   `json:"version"`
	ReleaseDate string   `json:"release_date"`
	Targets     []Target `json:"targets"`
}

// stableSemverTag 匹配稳定发布 tag vX.Y.Z（与发布 workflow 一致）。
var stableSemverTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// supportedTargets 是发布契约固定的六平台集合。
var supportedTargets = map[string]map[string]bool{
	"darwin":  {"amd64": true, "arm64": true},
	"linux":   {"amd64": true, "arm64": true},
	"windows": {"amd64": true, "arm64": true},
}

// NewManifest 构造一个已校验的发布清单。
func NewManifest(tag, releaseDate string, targets []Target) (*Manifest, error) {
	manifest := &Manifest{
		Schema:      ManifestSchema,
		Repository:  DefaultRepository,
		Tag:         tag,
		Version:     strings.TrimPrefix(tag, "v"),
		ReleaseDate: releaseDate,
		Targets:     targets,
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return manifest, nil
}

// Canonical 返回清单的规范 JSON 字节；签名对象针对这些字节计算。
func (m *Manifest) Canonical() ([]byte, error) {
	return canonicalJSON(m)
}

// Validate 校验清单的 schema、仓库、tag、版本、日期与六平台 target 集合。
func (m *Manifest) Validate() error {
	if m == nil {
		return fmt.Errorf("release manifest is nil")
	}
	if m.Schema != ManifestSchema {
		return fmt.Errorf("unsupported release manifest schema %q", m.Schema)
	}
	if m.Repository != DefaultRepository {
		return fmt.Errorf("release manifest repository %q does not match %q", m.Repository, DefaultRepository)
	}
	if !stableSemverTag.MatchString(m.Tag) {
		return fmt.Errorf("release manifest tag %q is not a stable vX.Y.Z tag", m.Tag)
	}
	if m.Version != strings.TrimPrefix(m.Tag, "v") {
		return fmt.Errorf("release manifest version %q does not match tag %q", m.Version, m.Tag)
	}
	if _, err := time.Parse("2006-01-02", m.ReleaseDate); err != nil {
		return fmt.Errorf("release manifest release_date %q is not YYYY-MM-DD: %w", m.ReleaseDate, err)
	}
	return validateTargets(m)
}

// ParseManifest 严格解析并校验发布清单字节：拒绝重复键、非规范编码、
// 尾随内容与任何语义违规。
func ParseManifest(raw []byte) (*Manifest, error) {
	var manifest Manifest
	if err := decodeStrictJSON(raw, &manifest); err != nil {
		return nil, fmt.Errorf("parse release manifest: %w", err)
	}
	canonical, err := manifest.Canonical()
	if err != nil {
		return nil, fmt.Errorf("canonicalize release manifest: %w", err)
	}
	if !bytes.Equal(canonical, raw) {
		return nil, fmt.Errorf("release manifest is not canonically encoded")
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func validateTargets(m *Manifest) error {
	if len(m.Targets) == 0 {
		return fmt.Errorf("release manifest has no targets")
	}
	if len(m.Targets) != 6 {
		return fmt.Errorf("release manifest has %d targets, want the fixed six-platform set", len(m.Targets))
	}
	seen := map[string]bool{}
	for _, target := range m.Targets {
		if !supportedTargets[target.GOOS][target.GOARCH] {
			return fmt.Errorf("release manifest has unsupported target %s/%s", target.GOOS, target.GOARCH)
		}
		platform := target.GOOS + "/" + target.GOARCH
		if seen[platform] {
			return fmt.Errorf("release manifest has duplicate target %s", platform)
		}
		seen[platform] = true
		if err := validateTarget(m, target); err != nil {
			return err
		}
	}
	for goos, architectures := range supportedTargets {
		for goarch := range architectures {
			if !seen[goos+"/"+goarch] {
				return fmt.Errorf("release manifest is missing target %s/%s", goos, goarch)
			}
		}
	}
	return nil
}

func validateTarget(m *Manifest, target Target) error {
	if !isLowercaseSHA256Hex(target.ArchiveSHA256) {
		return fmt.Errorf("release manifest target %s/%s has invalid archive SHA-256", target.GOOS, target.GOARCH)
	}
	if !isLowercaseSHA256Hex(target.BinarySHA256) {
		return fmt.Errorf("release manifest target %s/%s has invalid binary SHA-256", target.GOOS, target.GOARCH)
	}
	extension := ".tar.gz"
	binary := "javdb"
	if target.GOOS == "windows" {
		extension = ".zip"
		binary = "javdb.exe"
	}
	expectedArchive := "javdb-cli_" + m.Version + "_" + target.GOOS + "_" + target.GOARCH + extension
	if target.Archive != expectedArchive {
		return fmt.Errorf("release manifest target %s/%s archive %q does not match expected %q", target.GOOS, target.GOARCH, target.Archive, expectedArchive)
	}
	if target.Binary != binary {
		return fmt.Errorf("release manifest target %s/%s binary %q does not match expected %q", target.GOOS, target.GOARCH, target.Binary, binary)
	}
	return nil
}

func isLowercaseSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 32 {
		return false
	}
	return strings.ToLower(value) == value
}
