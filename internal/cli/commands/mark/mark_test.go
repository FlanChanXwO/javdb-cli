package mark

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

type markTrackingReader struct {
	data  []byte
	reads int
}

func (r *markTrackingReader) Read(p []byte) (int, error) {
	r.reads++
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestNewBuildsMarkCommand(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	if cmd.Name() != "mark" || cmd.Use != "mark NUMBER" {
		t.Fatalf("name=%q use=%q", cmd.Name(), cmd.Use)
	}
	for _, flag := range []string{"watched", "want", "score", "content", "id"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Fatalf("missing --%s", flag)
		}
	}
}

func TestNewRequiresExactlyOneFlagBeforeNetwork(t *testing.T) {
	streams := invocation.NewStreams(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs([]string{"ABC"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "specify exactly one of --watched or --want") {
		t.Fatalf("expected flag validation error, got %v", err)
	}
}

func TestMarkStatusValidationDoesNotReadStdin(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "missing status", args: nil},
		{name: "ambiguous status", args: []string{"--watched", "--want"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := &markTrackingReader{data: []byte("SSIS-589\n")}
			streams := invocation.NewStreams(reader, &bytes.Buffer{}, &bytes.Buffer{})
			cmd := New(&invocation.RootOptions{}, streams)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "specify exactly one of --watched or --want") {
				t.Fatalf("expected status validation error, got %v", err)
			}
			if reader.reads != 0 {
				t.Fatalf("stdin reads = %d, want 0", reader.reads)
			}
		})
	}
}

func TestMarkValidStatusLeavesMissingInputToRunner(t *testing.T) {
	reader := &markTrackingReader{}
	streams := invocation.NewStreams(reader, &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetArgs([]string{"--watched"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected runner to reject missing input")
	}
	if strings.Contains(err.Error(), "specify exactly one of --watched or --want") {
		t.Fatalf("status validation handled missing input: %v", err)
	}
	if reader.reads == 0 {
		t.Fatal("runner did not read stdin")
	}
}

func TestMarkValidNonTTYInputReachesBatchRunner(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	var searchedNumber string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.URL.Path == "/api/v2/search":
			searchedNumber = request.URL.Query().Get("q")
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"movies": []map[string]any{{"number": searchedNumber, "id": "id-" + searchedNumber}},
				},
			})
		case strings.HasSuffix(request.URL.Path, "/reviews"):
			_ = json.NewEncoder(writer).Encode(map[string]any{
				"success": true,
				"data":    map[string]any{"id": "review-1"},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	streams := invocation.NewStreams(strings.NewReader("SSIS-589\n"), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := New(&invocation.RootOptions{Host: server.URL}, streams)
	cmd.SetArgs([]string{"--want"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if searchedNumber != "SSIS-589" {
		t.Fatalf("search q = %q, want SSIS-589", searchedNumber)
	}
	if got := streams.Out.(*bytes.Buffer).String(); got != "SSIS-589\n" {
		t.Fatalf("output = %q, want forwarded movie line", got)
	}
}
