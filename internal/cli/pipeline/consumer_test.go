package pipeline

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	streams.OutIsTerminal = terminal
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
		{name: "ndjson batch", stdin: "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"a\"}\n{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"b\"}\n", terminal: false, arg: "", want: 2, wantKind: KindMovie},
		{name: "ndjson wrong kind", stdin: "{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"actor\",\"ref\":\"a\"}\n", terminal: false, arg: "", want: 1, wantKind: KindActor},
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
	err = consumer.RunInputs(streams, inputs, OutputNDJSON)
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

func TestConsumerSingleNDJSONLegacyShape(t *testing.T) {
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
	// 构造显式 --json 单输入路径：文本 ref（无 kind）才走 legacy shape。
	inputs := []Envelope{New("", "SSIS-589", "")}
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

func TestListProducerDefaultsToText(t *testing.T) {
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
		RowText: func(w io.Writer, _ io.Writer, items []map[string]any) error {
			for _, item := range items {
				if _, err := fmt.Fprintln(w, item["number"]); err != nil {
					return err
				}
			}
			return nil
		},
	}
	if err := producer.Execute(streams, false, false); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "SSIS-589\nHZGD-246\n" {
		t.Errorf("default output = %q", out.String())
	}
}

func TestBatchRunnerPipedInputDefaultsToText(t *testing.T) {
	streams, out := testStreams("SSIS-589\nHZGD-246\n", false)
	runner := &BatchRunner{
		Name:  "search",
		Kinds: []Kind{KindMovie},
		RunOne: func(_ *javdb.Client, _ context.Context, input Envelope) (Envelope, error) {
			return New(KindMovie, input.Ref, "id-"+input.Ref), nil
		},
		Legacy: func([]string) error {
			return testError("batch input must not use the single-item legacy path")
		},
	}
	if err := runner.Execute(streams, nil, false, false); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.String() != "SSIS-589\nHZGD-246\n" {
		t.Errorf("default output = %q", out.String())
	}
}

// TestConsumerSingleNDJSONEnvelopeStillChecksKind 单条 NDJSON 信封必须经过
// kind 校验并保留 id 语义（legacy 路径只服务于纯文本 ref 输入）。
func TestConsumerSingleNDJSONEnvelopeStillChecksKind(t *testing.T) {
	streams, out := testStreams("", false)
	consumer := &Consumer{
		Name:          "detail",
		AcceptedKinds: []Kind{KindMovie},
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			if input.Ref == "actor-ref" {
				return Envelope{}, testError("RunOne must not run for incompatible kinds")
			}
			// id 语义：使用信封的 id 而不是 ref。
			return New(KindMovie, input.Ref, input.ID).WithData(map[string]any{"movie_id": input.ID}), nil
		},
		LegacyJSON: func(w io.Writer, envelope Envelope) error {
			return testError("legacy JSON must not run for NDJSON envelopes")
		},
	}
	// 显式 --json 单条 actor 信封 → kind 校验失败（原位 error），不落 legacy。
	inputs := []Envelope{{Schema: Schema, Kind: KindActor, Ref: "actor-ref", ID: "actor-id"}}
	err := consumer.RunInputs(streams, inputs, OutputJSON)
	if err == nil {
		t.Fatal("expected kind mismatch failure summary")
	}
	if !strings.Contains(out.String(), `"kind":"error"`) || !strings.Contains(out.String(), "unsupported kind") {
		t.Errorf("kind error envelope missing: %s", out.String())
	}

	// 单条 movie 信封（带 id）+ 默认文本 → 走 RunOne 并保留 id。
	streams2, out2 := testStreams("", false)
	consumer2 := &Consumer{
		Name:          "detail",
		AcceptedKinds: []Kind{KindMovie},
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			return New(KindMovie, input.Ref, input.ID).WithData(map[string]any{"movie_id": input.ID}), nil
		},
	}
	movieInputs := []Envelope{{Schema: Schema, Kind: KindMovie, Ref: "SSIS-589", ID: "9DGB5X"}}
	if err := consumer2.RunInputs(streams2, movieInputs, OutputText); err != nil {
		t.Fatalf("RunInputs text: %v", err)
	}
	// 默认文本输出使用人类稳定引用（ref）；消费侧（RunOne）保留 id 语义。
	if strings.TrimSpace(out2.String()) != "SSIS-589" {
		t.Errorf("text output = %q, want ref SSIS-589", out2.String())
	}
}

// TestBatchRunnerTTYStdoutRoutesToLegacy TTY stdout 默认 OutputHuman，单项
// 纯文本 ref 输入走 legacy 人类文本路径而非 Writer 逐行 ref。
func TestBatchRunnerTTYStdoutRoutesToLegacy(t *testing.T) {
	streams, out := testStreams("", true)
	legacyCalled := false
	runner := &BatchRunner{
		Name:  "detail",
		Kinds: []Kind{KindMovie},
		RunOne: func(_ *javdb.Client, _ context.Context, input Envelope) (Envelope, error) {
			return New(KindMovie, input.Ref, "id"), nil
		},
		Legacy: func(args []string) error {
			legacyCalled = true
			_, err := fmt.Fprintln(out, "human-readable")
			return err
		},
	}
	// 单项纯文本 ref + TTY stdout → OutputHuman → legacy 路径。
	if err := runner.ExecuteWithInputs(streams, []Envelope{New("", "SSIS-589", "")}, OutputHuman); err != nil {
		t.Fatalf("ExecuteWithInputs: %v", err)
	}
	if !legacyCalled {
		t.Error("TTY stdout must route single text input through the legacy human path")
	}
}

// TestConsumerTextModeErrorsToStderr 文本/人类模式下失败项只写 stderr，
// 不向 stdout 输出任何内容（不伪造成功 ref）；成功项写 stdout。
func TestConsumerTextModeErrorsToStderr(t *testing.T) {
	streams, out := testStreams("a\nb\nc\n", false)
	errBuf := &bytes.Buffer{}
	streams.Err = errBuf
	consumer := &Consumer{
		Name:          "detail",
		AcceptedKinds: []Kind{KindMovie},
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			if input.Ref == "b" {
				return Envelope{}, testError("detail failed")
			}
			return New(KindMovie, input.Ref, "id-"+input.Ref), nil
		},
	}
	inputs, err := CollectInputs(bufio.NewReader(strings.NewReader("a\nb\nc\n")), streams, "", []Kind{KindMovie})
	if err != nil {
		t.Fatalf("CollectInputs: %v", err)
	}
	err = consumer.RunInputs(streams, inputs, OutputText)
	if err == nil || !strings.Contains(err.Error(), "1 of 3 items failed") {
		t.Fatalf("expected 1-of-3 failure summary, got %v", err)
	}
	// stdout 只有成功项的 ref，不含失败项。
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "c" {
		t.Errorf("stdout = %q, want only a and c", out.String())
	}
	// stderr 包含失败原因。
	if !strings.Contains(errBuf.String(), "detail failed") {
		t.Errorf("stderr = %q, want 'detail failed'", errBuf.String())
	}
}

// TestConsumerHumanModeRenderText OutputHuman 模式下成功项走 RenderText。
func TestConsumerHumanModeRenderText(t *testing.T) {
	streams, out := testStreams("", false)
	renderCalled := false
	consumer := &Consumer{
		Name:          "detail",
		AcceptedKinds: []Kind{KindMovie},
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			return New(KindMovie, input.Ref, "id"), nil
		},
		RenderText: func(w io.Writer, envelope Envelope) error {
			renderCalled = true
			_, err := fmt.Fprintf(w, ">> %s\n", envelope.Ref)
			return err
		},
	}
	if err := consumer.RunInputs(streams, []Envelope{New("", "SSIS-589", "")}, OutputHuman); err != nil {
		t.Fatalf("RunInputs: %v", err)
	}
	if !renderCalled {
		t.Error("OutputHuman must call RenderText for successful items")
	}
	if !strings.Contains(out.String(), ">> SSIS-589") {
		t.Errorf("output = %q, want '>> SSIS-589'", out.String())
	}
}
