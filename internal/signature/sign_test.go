package signature

import "testing"

func TestSignGolden(t *testing.T) {
	// Captured / verified against Python signature.py 2026-07-16.
	const want = "1784134914.lpw6vgqzsp.85b53cc0034eff62f361723615a3b8e3"
	got := Sign(1784134914)
	if got != want {
		t.Fatalf("Sign(1784134914)=\n  %q\nwant %q", got, want)
	}
}

func TestSignUsesCurrentTime(t *testing.T) {
	got := Sign(0)
	if got == "" || len(got) < 20 {
		t.Fatalf("unexpected empty/short signature: %q", got)
	}
}
