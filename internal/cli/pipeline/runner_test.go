package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestBatchRunnerEmptyInputUsesCommandName(t *testing.T) {
	streams, _ := testStreams("", true)
	runner := &BatchRunner{
		Name:  "detail",
		Kinds: []Kind{KindMovie},
		Legacy: func([]string) error {
			return fmt.Errorf("legacy path must not run")
		},
	}

	err := runner.Execute(streams, nil, false, false)
	if err == nil || err.Error() != "detail: input required" {
		t.Fatalf("error = %v, want detail: input required", err)
	}
}

func TestBatchRunnerEmptyInputWithoutNameUsesGenericMessage(t *testing.T) {
	streams, _ := testStreams("", true)
	runner := &BatchRunner{Kinds: []Kind{KindMovie}}

	err := runner.Execute(streams, nil, false, false)
	if err == nil || err.Error() != "input required" {
		t.Fatalf("error = %v, want input required", err)
	}
}

func TestProducerJSONRendererUsesAlreadyProducedEnvelopes(t *testing.T) {
	streams, out := testStreams("", false)
	events := []string{}
	producer := &Producer{
		Name: "lists",
		Produce: func(context.Context) ([]Envelope, error) {
			events = append(events, "produce")
			return []Envelope{New(KindList, "Favorites", "list-1")}, nil
		},
		RenderJSON: func(w io.Writer, envelopes []Envelope) error {
			events = append(events, "render")
			return json.NewEncoder(w).Encode(envelopes)
		},
		LegacyJSON: func(io.Writer) error {
			events = append(events, "legacy")
			return fmt.Errorf("legacy JSON must not run when RenderJSON is set")
		},
	}

	if err := producer.Execute(streams, false, true); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got, want := events, []string{"produce", "render"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	var envelopes []Envelope
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &envelopes); err != nil {
		t.Fatalf("JSON output = %q: %v", out.String(), err)
	}
	if len(envelopes) != 1 || envelopes[0].ID != "list-1" || envelopes[0].Ref != "Favorites" {
		t.Fatalf("JSON envelopes = %+v", envelopes)
	}
}

func TestProducerJSONWithoutRendererKeepsLegacyJSON(t *testing.T) {
	streams, out := testStreams("", false)
	produceCalls := 0
	legacyCalls := 0
	producer := &Producer{
		Name: "tags",
		Produce: func(context.Context) ([]Envelope, error) {
			produceCalls++
			return []Envelope{New(KindTag, "VR", "tag-1")}, nil
		},
		LegacyJSON: func(w io.Writer) error {
			legacyCalls++
			_, err := io.WriteString(w, `{"tags":["VR"]}`)
			return err
		},
	}

	if err := producer.Execute(streams, false, true); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if produceCalls != 1 {
		t.Fatalf("Produce calls = %d, want 1", produceCalls)
	}
	if legacyCalls != 1 {
		t.Fatalf("LegacyJSON calls = %d, want 1", legacyCalls)
	}
	if got, want := out.String(), `{"tags":["VR"]}`; strings.TrimSpace(got) != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestProducerNonJSONModesProduceOnce(t *testing.T) {
	tests := []struct {
		name     string
		terminal bool
		ndjson   bool
	}{
		{name: "text", terminal: false},
		{name: "human", terminal: true},
		{name: "ndjson", ndjson: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			streams, _ := testStreams("", tc.terminal)
			produceCalls := 0
			producer := &Producer{
				Name: "tags",
				Produce: func(context.Context) ([]Envelope, error) {
					produceCalls++
					return []Envelope{New(KindTag, "VR", "tag-1")}, nil
				},
				RenderText: func(w io.Writer, _ []Envelope) error {
					_, err := io.WriteString(w, "human\n")
					return err
				},
				LegacyJSON: func(io.Writer) error {
					return fmt.Errorf("LegacyJSON must not run")
				},
			}

			if err := producer.Execute(streams, tc.ndjson, false); err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if produceCalls != 1 {
				t.Fatalf("Produce calls = %d, want 1", produceCalls)
			}
		})
	}
}

func TestProducerRejectsMutuallyExclusiveOutputFlagsBeforeProduce(t *testing.T) {
	streams, _ := testStreams("", false)
	producer := &Producer{
		Name: "tags",
		Produce: func(context.Context) ([]Envelope, error) {
			t.Fatal("Produce must not run for mutually exclusive output flags")
			return nil, nil
		},
	}

	err := producer.Execute(streams, true, true)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error = %v, want mutually exclusive error", err)
	}
}
