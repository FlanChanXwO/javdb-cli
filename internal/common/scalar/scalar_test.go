package scalar

import (
	"encoding/json"
	"math"
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "nil", in: nil, want: ""},
		{name: "string", in: "SSIS-589", want: "SSIS-589"},
		{name: "json number", in: json.Number("12.50"), want: "12.50"},
		{name: "float", in: float64(12.5), want: "12.5"},
		{name: "integer", in: int32(12), want: "12"},
		{name: "unsigned integer", in: uint64(12), want: "12"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String(tt.in); got != tt.want {
				t.Fatalf("String(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestInt64(t *testing.T) {
	maxInt64 := int64(math.MaxInt64)
	tests := []struct {
		name string
		in   any
		want int64
		ok   bool
	}{
		{name: "nil", in: nil},
		{name: "json number", in: json.Number("42"), want: 42, ok: true},
		{name: "invalid json number", in: json.Number("42.5")},
		{name: "int", in: int(-42), want: -42, ok: true},
		{name: "int8", in: int8(-8), want: -8, ok: true},
		{name: "int16", in: int16(-16), want: -16, ok: true},
		{name: "int32", in: int32(-32), want: -32, ok: true},
		{name: "int64", in: maxInt64, want: maxInt64, ok: true},
		{name: "uint", in: uint(42), want: 42, ok: true},
		{name: "uint8", in: uint8(8), want: 8, ok: true},
		{name: "uint16", in: uint16(16), want: 16, ok: true},
		{name: "uint32", in: uint32(32), want: 32, ok: true},
		{name: "uint64", in: uint64(64), want: 64, ok: true},
		{name: "uintptr", in: uintptr(128), want: 128, ok: true},
		{name: "float32 truncates", in: float32(12.75), want: 12, ok: true},
		{name: "float64 truncates", in: float64(-12.75), want: -12, ok: true},
		{name: "string", in: "9223372036854775807", want: maxInt64, ok: true},
		{name: "invalid string", in: "12.5"},
		{name: "bool", in: true},
		{name: "uint64 overflow", in: uint64(maxInt64) + 1},
		{name: "float overflow", in: math.Pow(2, 63)},
		{name: "not a number", in: math.NaN()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Int64(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("Int64(%#v) = (%d, %t), want (%d, %t)", tt.in, got, ok, tt.want, tt.ok)
			}
		})
	}
}
