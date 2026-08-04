package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDownloadHelpDocumentsSinglePreviewImage(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"download", "--help"}, strings.NewReader(""), &out, &errb)
	if code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	for _, want := range []string{"--thumbnail", "--preview-image", "--preview-video", "only the first preview image"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("download help missing %q: %s", want, out.String())
		}
	}
}

func TestDownloadRequiresASelectedOutputBeforeNetworkAccess(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run([]string{"download", "WTEX-15"}, strings.NewReader(""), &out, &errb)
	if code == 0 {
		t.Fatal("download without output unexpectedly succeeded")
	}
	if !strings.Contains(errb.String(), "set at least one") {
		t.Fatalf("unexpected error: %s", errb.String())
	}
}
