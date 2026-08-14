package pipeline

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

var testJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00}

func TestEnvelopeValidate(t *testing.T) {
	valid := New(KindMovie, "SSIS-589", "9DGB5X")
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	for _, tc := range []struct {
		name     string
		envelope Envelope
	}{
		{name: "wrong schema", envelope: Envelope{Schema: "other", Kind: KindMovie, Ref: "x"}},
		{name: "unknown kind", envelope: Envelope{Schema: Schema, Kind: "ghost", Ref: "x"}},
		{name: "no ref and id", envelope: Envelope{Schema: Schema, Kind: KindMovie}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.envelope.Validate(); err == nil {
				t.Fatal("invalid envelope accepted")
			}
		})
	}
}

func TestDecodeNDJSON(t *testing.T) {
	envelope, err := DecodeNDJSON(`{"schema":"javdb.pipeline/v1","kind":"movie","ref":"SSIS-589","id":"9DGB5X"}`)
	if err != nil {
		t.Fatalf("DecodeNDJSON: %v", err)
	}
	if envelope.Ref != "SSIS-589" || envelope.ID != "9DGB5X" {
		t.Errorf("envelope = %+v", envelope)
	}

	// meta/data 保留。
	envelope, err = DecodeNDJSON(`{"schema":"javdb.pipeline/v1","kind":"movie","ref":"x","meta":{"source":"builtin"}}`)
	if err != nil {
		t.Fatalf("DecodeNDJSON with meta: %v", err)
	}
	if envelope.Meta["source"] != "builtin" {
		t.Errorf("meta lost: %+v", envelope.Meta)
	}

	for _, tc := range []struct {
		name string
		line string
	}{
		{name: "empty", line: "  "},
		{name: "not json", line: "plain text"},
		{name: "array", line: `[1,2]`},
		{name: "wrong schema", line: `{"schema":"other","kind":"movie","ref":"x"}`},
		{name: "trailing data", line: `{"schema":"javdb.pipeline/v1","kind":"movie","ref":"x"} extra`},
		{name: "no ref", line: `{"schema":"javdb.pipeline/v1","kind":"movie"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeNDJSON(tc.line); err == nil {
				t.Fatal("DecodeNDJSON accepted invalid line")
			}
		})
	}
}

func TestParseBatchTextAndNDJSON(t *testing.T) {
	text, err := ParseBatch([]byte("SSIS-589\nHZGD-246\n"), KindMovie)
	if err != nil {
		t.Fatalf("ParseBatch text: %v", err)
	}
	if len(text) != 2 || text[0].Ref != "SSIS-589" || text[1].Kind != KindMovie {
		t.Errorf("text batch = %+v", text)
	}

	ndjson, err := ParseBatch([]byte("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"a\"}\n{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"b\"}\n"), KindNDJSONInput)
	if err != nil {
		t.Fatalf("ParseBatch ndjson: %v", err)
	}
	if len(ndjson) != 2 || ndjson[0].Ref != "a" || ndjson[1].Ref != "b" {
		t.Errorf("ndjson batch = %+v", ndjson)
	}

	// 混合/非法行必须带行号报错。
	if _, err := ParseBatch([]byte("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"a\"}\ngarbage\n"), KindNDJSONInput); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("mixed batch error = %v", err)
	}
}

func TestClassifyImageNDJSONText(t *testing.T) {
	// 图片 magic 优先。
	classification, content, err := Classify(bufio.NewReader(bytes.NewReader(testJPEG)))
	if err != nil || classification != ClassificationImage {
		t.Fatalf("image classification = %v err=%v", classification, err)
	}
	_ = content

	// NDJSON。
	ndjson := []byte("{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"a\"}\n{\"schema\":\"javdb.pipeline/v1\",\"kind\":\"movie\",\"ref\":\"b\"}\n")
	classification, content, err = Classify(bufio.NewReader(bytes.NewReader(ndjson)))
	if err != nil || classification != ClassificationNDJSON {
		t.Fatalf("ndjson classification = %v err=%v", classification, err)
	}
	if !bytes.Equal(content, ndjson) {
		t.Error("ndjson content not preserved")
	}

	// 文本。
	classification, content, err = Classify(bufio.NewReader(strings.NewReader("SSIS-589\nHZGD-246\n")))
	if err != nil || classification != ClassificationText {
		t.Fatalf("text classification = %v err=%v", classification, err)
	}

	// 空输入是文本。
	classification, _, err = Classify(bufio.NewReader(strings.NewReader("")))
	if err != nil || classification != ClassificationText {
		t.Fatalf("empty classification = %v err=%v", classification, err)
	}
}

func TestResolveOutputMode(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ndjson        bool
		json          bool
		outIsTerminal bool
		want          OutputMode
		wantError     bool
	}{
		{name: "default tty", outIsTerminal: true, want: OutputHuman},
		{name: "default non-tty", outIsTerminal: false, want: OutputText},
		{name: "explicit ndjson tty", ndjson: true, outIsTerminal: true, want: OutputNDJSON},
		{name: "explicit ndjson non-tty", ndjson: true, outIsTerminal: false, want: OutputNDJSON},
		{name: "explicit json tty", json: true, outIsTerminal: true, want: OutputJSON},
		{name: "explicit json non-tty", json: true, outIsTerminal: false, want: OutputJSON},
		{name: "ndjson and json", ndjson: true, json: true, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mode, err := ResolveOutputMode(tc.ndjson, tc.json, tc.outIsTerminal)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected mutual exclusion error")
				}
				return
			}
			if err != nil || mode != tc.want {
				t.Fatalf("mode = %v err = %v, want %v", mode, err, tc.want)
			}
		})
	}
}

// TestWriterHumanFallsBackToText Writer 在 OutputHuman 模式下退化为逐行 ref，
// 与 OutputText 行为一致；命令级人类渲染在进入 Writer 前已分流。
func TestWriterHumanFallsBackToText(t *testing.T) {
	envelope := New(KindMovie, "SSIS-589", "9DGB5X")
	var out bytes.Buffer
	writer := NewWriter(&out, OutputHuman)
	if err := writer.Write(envelope); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "SSIS-589" {
		t.Errorf("human fallback output = %q, want SSIS-589", out.String())
	}
}

func TestWriterModes(t *testing.T) {
	envelope := New(KindMovie, "SSIS-589", "9DGB5X")

	// NDJSON：每行一个信封。
	var ndjsonOut bytes.Buffer
	ndjsonWriter := NewWriter(&ndjsonOut, OutputNDJSON)
	if err := ndjsonWriter.Write(envelope); err != nil {
		t.Fatal(err)
	}
	if err := ndjsonWriter.Write(envelope.WithData(map[string]any{"n": 1})); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(ndjsonOut.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("ndjson lines = %d", len(lines))
	}
	var parsed Envelope
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil || parsed.Schema != Schema {
		t.Fatalf("ndjson line invalid: %v", err)
	}

	// Text：逐行 ref。
	var textOut bytes.Buffer
	textWriter := NewWriter(&textOut, OutputText)
	if err := textWriter.Write(envelope); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(textOut.String()) != "SSIS-589" {
		t.Errorf("text output = %q", textOut.String())
	}

	// JSON：单项对象，多项数组。
	var singleOut bytes.Buffer
	singleWriter := NewWriter(&singleOut, OutputJSON)
	if err := singleWriter.Write(envelope); err != nil {
		t.Fatal(err)
	}
	if err := singleWriter.Finish(); err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(singleOut.Bytes(), &object); err != nil || object["ref"] != "SSIS-589" {
		t.Fatalf("single json output = %q err=%v", singleOut.String(), err)
	}

	var multiOut bytes.Buffer
	multiWriter := NewWriter(&multiOut, OutputJSON)
	if err := multiWriter.Write(envelope); err != nil {
		t.Fatal(err)
	}
	if err := multiWriter.Write(New(KindMovie, "HZGD-246", "")); err != nil {
		t.Fatal(err)
	}
	if err := multiWriter.Finish(); err != nil {
		t.Fatal(err)
	}
	var array []map[string]any
	if err := json.Unmarshal(multiOut.Bytes(), &array); err != nil || len(array) != 2 {
		t.Fatalf("multi json output = %q err=%v", multiOut.String(), err)
	}
}

func TestRunBatchOrderErrorsAndExit(t *testing.T) {
	var out bytes.Buffer
	writer := NewWriter(&out, OutputNDJSON)
	inputs := []Envelope{
		New(KindMovie, "a", ""),
		New(KindMovie, "b", ""),
		New(KindMovie, "c", ""),
	}
	failures, err := RunBatch(writer, inputs, "test", func(input Envelope) (Envelope, error) {
		if input.Ref == "b" {
			return Envelope{}, testError("boom")
		}
		return New(KindMovie, input.Ref+"-ok", ""), nil
	})
	if err != nil {
		t.Fatalf("RunBatch: %v", err)
	}
	if failures != 1 {
		t.Fatalf("failures = %d, want 1", failures)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d, want 3 (all items emitted)", len(lines))
	}
	// 原位错误信封出现在失败项的位置。
	var second Envelope
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatal(err)
	}
	if second.Kind != KindError || second.Ref != "b" || second.Data["message"] != "boom" {
		t.Errorf("in-place error envelope = %+v", second)
	}
}

type testError string

func (e testError) Error() string { return string(e) }

// TestTagKindRoundTrip tags 自产的 tag 信封必须能通过管道解码。
func TestTagKindRoundTrip(t *testing.T) {
	line := `{"schema":"javdb.pipeline/v1","kind":"tag","ref":"VR","id":"t-1","data":{"name_zh":"VR"}}`
	envelope, err := DecodeNDJSON(line)
	if err != nil {
		t.Fatalf("DecodeNDJSON tag envelope: %v", err)
	}
	if envelope.Kind != KindTag || envelope.ID != "t-1" {
		t.Errorf("tag envelope = %+v", envelope)
	}
}
