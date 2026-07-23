package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestUpdateCommandRejectsJSONInstallOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"update", "--json"}, strings.NewReader(""), &stdout, &stderr); code == 0 {
		t.Fatal("update --json without --check unexpectedly succeeded")
	}
	if !strings.Contains(stderr.String(), "--json is only supported with --check") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestUpdateCommandIsRegistered(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"update", "--help"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("update help failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Check for or install updates") {
		t.Fatalf("update help = %q", stdout.String())
	}
}
