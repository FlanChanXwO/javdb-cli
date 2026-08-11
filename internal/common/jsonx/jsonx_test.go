package jsonx

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestObjectArray(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want []map[string]any
	}{
		{name: "nil", want: nil},
		{name: "empty raw", raw: json.RawMessage{}, want: nil},
		{name: "null", raw: json.RawMessage("null"), want: nil},
		{name: "empty array", raw: json.RawMessage("[]"), want: []map[string]any{}},
		{
			name: "objects",
			raw:  json.RawMessage(`[{"id":"one","count":2}]`),
			want: []map[string]any{{"id": "one", "count": float64(2)}},
		},
		{
			name: "null element preserved",
			raw:  json.RawMessage(`[null,{"id":"two"}]`),
			want: []map[string]any{nil, {"id": "two"}},
		},
		{name: "object is not array", raw: json.RawMessage(`{"id":"one"}`), want: nil},
		{name: "invalid", raw: json.RawMessage(`[`)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ObjectArray(tt.raw); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ObjectArray(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestObjectSlice(t *testing.T) {
	type object struct {
		ID string `json:"id"`
	}

	mapSlice := []map[string]any{{"id": "one"}}
	tests := []struct {
		name string
		in   any
		want []map[string]any
	}{
		{name: "nil", want: nil},
		{name: "map slice", in: mapSlice, want: mapSlice},
		{
			name: "any slice filters non maps",
			in:   []any{map[string]any{"id": "one"}, "ignored", nil, map[string]any{"id": "two"}},
			want: []map[string]any{{"id": "one"}, {"id": "two"}},
		},
		{name: "struct slice", in: []object{{ID: "one"}}, want: []map[string]any{{"id": "one"}}},
		{name: "empty struct slice", in: []object{}, want: []map[string]any{}},
		{name: "object is not slice", in: map[string]any{"id": "one"}, want: nil},
		{name: "unsupported value", in: make(chan int), want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ObjectSlice(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ObjectSlice(%#v) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}
}

func TestRawString(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "empty", raw: json.RawMessage{}, want: ""},
		{name: "quoted", raw: json.RawMessage(`"2026-08-10T00:00:00Z"`), want: "2026-08-10T00:00:00Z"},
		{name: "chinese quoted", raw: json.RawMessage(`"你好"`), want: "你好"},
		{name: "number raw", raw: json.RawMessage(`123`), want: "123"},
		// 不 unescape：内层转义保留为原始字节。
		{name: "escaped quote stays raw", raw: json.RawMessage(`"a\"b"`), want: `a\"b`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RawString(tt.raw); got != tt.want {
				t.Fatalf("RawString(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestMarshalLine(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "nil", value: nil, want: "null\n"},
		{name: "map sorted keys no html escape", value: map[string]any{"title": "A<B>&C", "chinese": "你好"}, want: "{\"chinese\":\"你好\",\"title\":\"A<B>&C\"}\n"},
		{name: "array", value: []int{1, 2}, want: "[1,2]\n"},
		{name: "scalar", value: "dev", want: "\"dev\"\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalLine(tt.value)
			if err != nil {
				t.Fatalf("MarshalLine(%v) error = %v", tt.value, err)
			}
			if string(got) != tt.want {
				t.Fatalf("MarshalLine(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestMarshalLineExactlyOneTrailingNewline(t *testing.T) {
	got, err := MarshalLine(map[string]string{"a": "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 || got[len(got)-1] != '\n' {
		t.Fatalf("MarshalLine missing trailing newline: %q", got)
	}
	if bytes.Count(got, []byte{'\n'}) != 1 {
		t.Fatalf("MarshalLine newline count = %d, want exactly 1: %q", bytes.Count(got, []byte{'\n'}), got)
	}
}

func TestMarshalLinePropagatesEncodingError(t *testing.T) {
	// 包含无法编码的值时必须返回错误，不吞掉。
	if _, err := MarshalLine(map[string]any{"x": make(chan int)}); err == nil {
		t.Fatal("MarshalLine accepted an unencodable value")
	}
}
