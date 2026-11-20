package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestDiagnosticFormatsAndBatchOwnership(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			var out bytes.Buffer
			logger, err := New(&out, "warning", format)
			if err != nil {
				t.Fatal(err)
			}
			logger = logger.With("command", "javdb actor")
			if err := Emit(logger, slog.LevelInfo, "hidden"); err != nil {
				t.Fatal(err)
			}
			if err := Emit(logger, slog.LevelWarn, "warning message\ncontinued"); err != nil {
				t.Fatal(err)
			}
			if err := Failure(logger, errors.New("upstream detail\nnext line")); err != nil {
				t.Fatal(err)
			}
			if strings.Count(out.String(), "\n") != 2 {
				t.Fatalf("each record must occupy one line: %s", out.String())
			}
			if format == "json" {
				var first struct{ Level string }
				if err := json.Unmarshal(bytes.Split(out.Bytes(), []byte{'\n'})[0], &first); err != nil || first.Level != "WARNING" {
					t.Fatalf("full warning level missing: %s", out.String())
				}
			} else if !strings.Contains(out.String(), "[WARNING]") || !strings.Contains(out.String(), "logging/logging_test.go:") {
				t.Fatalf("full level or caller missing: %s", out.String())
			}
			out.Reset()
			if err := Failure(logger, &Reported{Failed: 1, Total: 1}); err != nil || out.Len() != 0 {
				t.Fatalf("single-item failure repeated: %s", out.String())
			}
			if err := Failure(logger, &Reported{Failed: 1, Total: 2}); err != nil || out.Len() == 0 {
				t.Fatalf("batch summary missing: %v", err)
			}
		})
	}
}
