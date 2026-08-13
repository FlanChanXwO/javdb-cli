package pipeline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

func testStreams(stdin string, terminal bool) (*invocation.Streams, *bytes.Buffer) {
	out := &bytes.Buffer{}
	streams := invocation.NewStreams(strings.NewReader(stdin), out, &bytes.Buffer{})
	streams.InIsTerminal = terminal
	return streams, out
}

func TestCollectInputsMatrix(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stdin    string
		terminal bool
		arg      string
		want     int
		wantKind Kind
		wantErr  bool
	}{
		{name: "positional arg", stdin: "", terminal: true, arg: "SSIS-589", want: 1, wantKind: ""},
		{name: "tty no input", stdin: "", terminal: true, arg: "", want: 0},
		{name: "text batch", stdin: "SSIS-589\nHZGD-246\n", terminal: false, arg: "", want: 2, wantKind: KindMovie},
		{name: "jsonl batch", stdin: "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"a\"}\n{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"b\"}\n", terminal: false, arg: "", want: 2, wantKind: KindMovie},
		{name: "jsonl wrong kind", stdin: "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"actor\",\"ref\":\"a\"}\n", terminal: false, arg: "", want: 1, wantKind: KindActor},
		{name: "ambiguous arg and stdin", stdin: "data\n", terminal: false, arg: "x", wantErr: true},
		{name: "image rejected", stdin: string([]byte{0xFF, 0xD8, 0xFF}), terminal: false, arg: "", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			streams := invocation.NewStreams(strings.NewReader(tc.stdin), &bytes.Buffer{}, &bytes.Buffer{})
			streams.InIsTerminal = tc.terminal
			inputs, err := CollectInputs(bufio.NewReader(streams.In), streams, tc.arg, []Kind{KindMovie})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("CollectInputs: %v", err)
			}
			if len(inputs) != tc.want {
				t.Fatalf("inputs = %d, want %d", len(inputs), tc.want)
			}
			if tc.want > 0 && tc.wantKind != "" && inputs[0].Kind != tc.wantKind {
				t.Errorf("kind = %q, want %q", inputs[0].Kind, tc.wantKind)
			}
		})
	}
}

func TestConsumerKindCheckingAndInPlaceErrors(t *testing.T) {
	streams, out := testStreams(`{"schema":"javdb.pipeline/v1","kind":"movie","ref":"a"}
{"schema":"javdb.pipeline/v1","kind":"actor","ref":"b"}
{"schema":"javdb.pipeline/v1","kind":"movie","ref":"c"}
`, false)
	consumer := &Consumer{
		Name:          "detail",
		AcceptedKinds: []Kind{KindMovie},
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			if input.Ref == "c" {
				return Envelope{}, testError("detail failed")
			}
			return New(KindMovie, input.Ref, "id-"+input.Ref).WithData(map[string]any{"movie": map[string]any{"number": input.Ref}}), nil
		},
	}
	inputs, err := CollectInputs(bufio.NewReader(streams.In), streams, "", []Kind{KindMovie})
	if err != nil {
		t.Fatalf("CollectInputs: %v", err)
	}
	err = consumer.RunInputs(streams, inputs, OutputJSONL)
	if err == nil || !strings.Contains(err.Error(), "2 of 3 items failed") {
		t.Fatalf("expected 2-of-3 failure summary, got %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines = %d, want 3", len(lines))
	}
	var kinds []string
	for _, line := range lines {
		var envelope Envelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			t.Fatal(err)
		}
		kinds = append(kinds, string(envelope.Kind))
	}
	// 顺序保持：movie、kind 错误（原位）、movie 执行错误（原位）。
	if kinds[0] != "movie" || kinds[1] != "error" || kinds[2] != "error" {
		t.Errorf("kinds = %v", kinds)
	}
	var second Envelope
	_ = json.Unmarshal([]byte(lines[1]), &second)
	if second.Data["code"] != "kind" {
		t.Errorf("kind error envelope = %+v", second)
	}
}

func TestConsumerSingleJSONLegacyShape(t *testing.T) {
	streams, out := testStreams("", true)
	legacyCalled := false
	consumer := &Consumer{
		Name:          "detail",
		AcceptedKinds: []Kind{KindMovie},
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			return New(KindMovie, input.Ref, "id"), nil
		},
		LegacyJSON: func(w io.Writer, envelope Envelope) error {
			legacyCalled = true
			_, err := io.WriteString(w, `{"legacy":true}`)
			return err
		},
	}
	// 构造显式 --json 单输入路径：直接调用 RunInputs。
	inputs := []Envelope{New(KindMovie, "SSIS-589", "")}
	if err := consumer.RunInputs(streams, inputs, OutputJSON); err != nil {
		t.Fatalf("RunInputs: %v", err)
	}
	if !legacyCalled {
		t.Error("single-input --json must use the legacy shape")
	}
	if out.String() != `{"legacy":true}` {
		t.Errorf("output = %q", out.String())
	}
}

func TestListProducerJSONLOutput(t *testing.T) {
	streams, out := testStreams("", false)
	producer := &ListProducer{
		Name: "watched",
		ClientFactory: func() (*javdb.Client, error) {
			return nil, nil
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			return []map[string]any{
				{"number": "SSIS-589", "id": "id-a"},
				{"number": "HZGD-246", "id": "id-b"},
			}, nil
		},
		JSON: func(items []map[string]any) (map[string]any, error) {
			return map[string]any{"movies": items}, nil
		},
	}
	if err := producer.Execute(streams, false, false, false); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("jsonl lines = %d", len(lines))
	}
	var first Envelope
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatal(err)
	}
	if first.Kind != KindMovie || first.Ref != "SSIS-589" || first.ID != "id-a" {
		t.Errorf("envelope = %+v", first)
	}
}
