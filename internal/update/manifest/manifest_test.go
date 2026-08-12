package manifest

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewManifestFillsSchemaAndRepository(t *testing.T) {
	manifest, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	if manifest.Schema != ManifestSchema {
		t.Errorf("schema = %q, want %q", manifest.Schema, ManifestSchema)
	}
	if manifest.Repository != DefaultRepository {
		t.Errorf("repository = %q, want %q", manifest.Repository, DefaultRepository)
	}
	if manifest.Version != "0.7.0" {
		t.Errorf("version = %q, want %q", manifest.Version, "0.7.0")
	}
}

func TestNewManifestRejectsInvalidArguments(t *testing.T) {
	valid := validTargets()
	for _, tc := range []struct {
		name    string
		tag     string
		date    string
		targets []Target
	}{
		{name: "tag without v", tag: "0.7.0", date: "2026-08-12", targets: valid},
		{name: "prerelease tag", tag: "v0.7.0-rc.1", date: "2026-08-12", targets: valid},
		{name: "empty tag", tag: "", date: "2026-08-12", targets: valid},
		{name: "empty release date", tag: "v0.7.0", date: "", targets: valid},
		{name: "invalid release date", tag: "v0.7.0", date: "12.08.2026", targets: valid},
		{name: "no targets", tag: "v0.7.0", date: "2026-08-12", targets: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewManifest(tc.tag, tc.date, tc.targets); err == nil {
				t.Fatal("NewManifest accepted invalid input")
			}
		})
	}

	duplicate := append([]Target{}, valid...)
	duplicate = append(duplicate, valid[0])
	if _, err := NewManifest("v0.7.0", "2026-08-12", duplicate); err == nil {
		t.Fatal("NewManifest accepted duplicate targets")
	}
}

func TestManifestCanonicalEncodingIsDeterministic(t *testing.T) {
	first, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	second, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	firstBytes, err := first.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	secondBytes, err := second.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Error("canonical encoding is not deterministic")
	}
	if len(firstBytes) == 0 || firstBytes[len(firstBytes)-1] == '\n' {
		t.Error("canonical encoding must be compact JSON without a trailing newline")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	expected, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	raw, err := expected.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	parsed, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if parsed.Tag != expected.Tag || parsed.Version != expected.Version || parsed.ReleaseDate != expected.ReleaseDate {
		t.Errorf("round trip changed manifest fields: %+v", parsed)
	}
	if len(parsed.Targets) != len(expected.Targets) {
		t.Fatalf("round trip changed target count: %d != %d", len(parsed.Targets), len(expected.Targets))
	}
}

func TestParseManifestRejectsDuplicateFields(t *testing.T) {
	raw, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	canonical, err := raw.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	index := bytes.Index(canonical, []byte(`"repository":`))
	if index < 0 {
		t.Fatal("cannot locate repository field for injection")
	}
	// 在 canonical 文档顶层注入第二个 repository 字段，构成重复键。
	injected := make([]byte, 0, len(canonical)+64)
	injected = append(injected, canonical[:index]...)
	injected = append(injected, `"repository":"injected",`...)
	injected = append(injected, canonical[index:]...)
	if _, err := ParseManifest(injected); err == nil {
		t.Fatal("ParseManifest accepted duplicate top-level field")
	}
}

func TestParseManifestRejectsNonCanonicalEncoding(t *testing.T) {
	raw, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	canonical, err := raw.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, canonical, "", "  "); err != nil {
		t.Fatalf("indent JSON: %v", err)
	}
	if _, err := ParseManifest(pretty.Bytes()); err == nil {
		t.Error("ParseManifest accepted pretty-printed JSON")
	}

	if _, err := ParseManifest(reorderFields(t, canonical, "repository", "schema")); err == nil {
		t.Error("ParseManifest accepted reordered JSON fields")
	}

	whitespace := bytes.Replace(canonical, []byte(`{"schema"`), []byte(`{ "schema"`), 1)
	if _, err := ParseManifest(whitespace); err == nil {
		t.Error("ParseManifest accepted non-minimal whitespace")
	}
}

func TestParseManifestRejectsMalformedInput(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{name: "empty", raw: nil},
		{name: "not json", raw: []byte("not json")},
		{name: "trailing data", raw: append([]byte(`{}`), []byte(` {}`)...)},
		{name: "top level array", raw: []byte(`[]`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseManifest(tc.raw); err == nil {
				t.Fatal("ParseManifest accepted malformed input")
			}
		})
	}
}

func TestParseManifestRejectsSemanticViolations(t *testing.T) {
	raw, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	canonical, err := raw.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}

	for _, tc := range []struct {
		name  string
		field string
		value string
	}{
		{name: "wrong schema", field: "schema", value: `"javdb.release-manifest/v9"`},
		{name: "wrong repository", field: "repository", value: `"evil/javdb-cli"`},
		{name: "tag without v", field: "tag", value: `"0.7.0"`},
		{name: "prerelease tag", field: "tag", value: `"v0.7.0-rc.1"`},
		{name: "version does not match tag", field: "version", value: `"0.7.1"`},
		{name: "invalid release date", field: "release_date", value: `"2026-13-99"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mutated := mutateField(t, canonical, tc.field, tc.value)
			if _, err := ParseManifest(mutated); err == nil {
				t.Fatal("ParseManifest accepted semantic violation")
			}
		})
	}
}

func TestParseManifestRejectsIncompleteTargets(t *testing.T) {
	raw, err := NewManifest("v0.7.0", "2026-08-12", validTargets())
	if err != nil {
		t.Fatalf("NewManifest: %v", err)
	}
	canonical, err := raw.Canonical()
	if err != nil {
		t.Fatalf("Canonical: %v", err)
	}
	withoutOne := removeFirstTarget(t, canonical)
	if _, err := ParseManifest(withoutOne); err == nil {
		t.Error("ParseManifest accepted manifest with missing target")
	}
}

func TestNewManifestRejectsInvalidTargetFields(t *testing.T) {
	upper := validTargets()
	upper[0].ArchiveSHA256 = strings.ToUpper(upper[0].ArchiveSHA256)
	if _, err := NewManifest("v0.7.0", "2026-08-12", upper); err == nil {
		t.Fatal("NewManifest accepted uppercase SHA-256")
	}

	wrongExt := validTargets()
	wrongExt[0].Archive = "javdb-cli_0.7.0_darwin_amd64.rar"
	if _, err := NewManifest("v0.7.0", "2026-08-12", wrongExt); err == nil {
		t.Fatal("NewManifest accepted wrong archive extension")
	}

	wrongBinary := validTargets()
	wrongBinary[0].Binary = "javdb-cli"
	if _, err := NewManifest("v0.7.0", "2026-08-12", wrongBinary); err == nil {
		t.Fatal("NewManifest accepted wrong binary name")
	}

	unsupported := validTargets()
	unsupported[0].GOARCH = "riscv64"
	if _, err := NewManifest("v0.7.0", "2026-08-12", unsupported); err == nil {
		t.Fatal("NewManifest accepted unsupported GOARCH")
	}

	shortHash := validTargets()
	shortHash[0].BinarySHA256 = strings.Repeat("a", 63)
	if _, err := NewManifest("v0.7.0", "2026-08-12", shortHash); err == nil {
		t.Fatal("NewManifest accepted short SHA-256")
	}
}

// mutateField 替换 canonical JSON 文档中某字段的值并返回新文档。
func mutateField(t *testing.T, raw []byte, field, value string) []byte {
	t.Helper()
	needle := []byte(`"` + field + `":`)
	index := bytes.Index(raw, needle)
	if index < 0 {
		t.Fatalf("cannot locate field %q", field)
	}
	end := bytes.IndexByte(raw[index+len(needle):], ',')
	if end < 0 {
		t.Fatalf("cannot locate end of field %q", field)
	}
	end += index + len(needle)
	mutated := make([]byte, 0, len(raw))
	mutated = append(mutated, raw[:index+len(needle)]...)
	mutated = append(mutated, value...)
	mutated = append(mutated, raw[end:]...)
	return mutated
}

// removeFirstTarget 删除 canonical 文档中的第一个 target 对象。
func removeFirstTarget(t *testing.T, raw []byte) []byte {
	t.Helper()
	start := bytes.Index(raw, []byte(`{"goos":`))
	if start < 0 {
		t.Fatal("cannot locate target object")
	}
	depth := 0
	for index := start; index < len(raw); index++ {
		switch raw[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				removed := make([]byte, 0, len(raw))
				removed = append(removed, raw[:start]...)
				removed = append(removed, raw[index+1:]...)
				return removed
			}
		}
	}
	t.Fatal("cannot find end of target object")
	return nil
}

// reorderFields 交换 canonical 文档中两个字段的文本位置。
func reorderFields(t *testing.T, raw []byte, first, second string) []byte {
	t.Helper()
	locate := func(name string) (start, end int) {
		start = bytes.Index(raw, []byte(`"`+name+`":`))
		if start < 0 {
			t.Fatalf("cannot locate field %q", name)
		}
		end = bytes.IndexByte(raw[start:], ',')
		if end < 0 {
			t.Fatalf("cannot locate end of field %q", name)
		}
		return start, start + end
	}
	firstStart, firstEnd := locate(first)
	secondStart, secondEnd := locate(second)
	lowStart, highEnd := firstStart, firstEnd
	lowEnd, highStart := secondEnd, secondStart
	if firstStart > secondStart {
		lowStart, lowEnd = secondStart, secondEnd
		highStart, highEnd = firstStart, firstEnd
	}
	reordered := make([]byte, 0, len(raw))
	reordered = append(reordered, raw[:lowStart]...)
	reordered = append(reordered, raw[highStart:highEnd]...)
	reordered = append(reordered, ',')
	reordered = append(reordered, raw[lowEnd:highStart]...)
	reordered = append(reordered, raw[lowStart:lowEnd]...)
	reordered = append(reordered, ',')
	reordered = append(reordered, raw[highEnd:]...)
	return reordered
}
