package codec

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
)

func TestRawStringAndParseID(t *testing.T) {
	data := map[string]json.RawMessage{
		"name": json.RawMessage(`"alice"`),
		"id":   json.RawMessage(`123.75`),
	}
	if got := RawString(data, "name"); got != "alice" {
		t.Fatalf("RawString(name) = %q", got)
	}
	if got := RawString(data, "id"); got != "123.75" {
		t.Fatalf("RawString(id) = %q", got)
	}
	if got := RawString(data, "missing"); got != "" {
		t.Fatalf("RawString(missing) = %q", got)
	}

	tests := []struct {
		name string
		raw  json.RawMessage
		want int64
		ok   bool
	}{
		{name: "integer", raw: json.RawMessage(`123`), want: 123, ok: true},
		{name: "string", raw: json.RawMessage(`"456"`), want: 456, ok: true},
		{name: "float truncates", raw: json.RawMessage(`789.9`), want: 789, ok: true},
		{name: "zero", raw: json.RawMessage(`0`)},
		{name: "invalid", raw: json.RawMessage(`"not-a-number"`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseID(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("ParseID(%q) = (%d, %t), want (%d, %t)", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestUserIDFromMapPrefersIDAndNameFallback(t *testing.T) {
	data := map[string]json.RawMessage{
		"uid":   json.RawMessage(`42`),
		"email": json.RawMessage(`"alice@example.test"`),
	}
	id, name, ok := UserIDFromMap(data)
	if !ok || id != 42 || name != "alice@example.test" {
		t.Fatalf("UserIDFromMap = (%d, %q, %t)", id, name, ok)
	}
}

func TestUserIDFromJWT(t *testing.T) {
	payload, err := json.Marshal(map[string]any{"sub": "73", "username": "alice"})
	if err != nil {
		t.Fatal(err)
	}
	segment := base64.RawURLEncoding.EncodeToString(payload)
	id, name, ok := UserIDFromJWT("header." + segment + ".signature")
	if !ok || id != 73 || name != "alice" {
		t.Fatalf("UserIDFromJWT = (%d, %q, %t)", id, name, ok)
	}

	nested, err := json.Marshal(map[string]any{"user": map[string]any{"id": float64(88), "name": "nested"}})
	if err != nil {
		t.Fatal(err)
	}
	segment = base64.RawURLEncoding.EncodeToString(nested)
	id, name, ok = UserIDFromJWT("header." + segment + ".signature")
	if !ok || id != 88 || name != "nested" {
		t.Fatalf("nested UserIDFromJWT = (%d, %q, %t)", id, name, ok)
	}

	if _, _, ok := UserIDFromJWT("not-a-jwt"); ok {
		t.Fatal("malformed JWT unexpectedly resolved")
	}
}

func TestObjectCodecsForwardSharedSemantics(t *testing.T) {
	raw := json.RawMessage(`[{"id":"one"}]`)
	want := []map[string]any{{"id": "one"}}
	if got := ObjectArray(raw); !reflect.DeepEqual(got, want) {
		t.Fatalf("ObjectArray = %#v, want %#v", got, want)
	}
	if got := ObjectSlice([]any{map[string]any{"id": "one"}, "ignored"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ObjectSlice = %#v, want %#v", got, want)
	}
}
