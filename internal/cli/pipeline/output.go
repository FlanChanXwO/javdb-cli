package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// OutputMode 是命令的机器输出格式。
type OutputMode int

const (
	// OutputAuto 由 TTY 状态决定：TTY 人类文本，非 TTY JSONL。
	OutputAuto OutputMode = iota
	// OutputText 强制逐行 ref 文本。
	OutputText
	// OutputJSONL 强制逐条信封 JSONL。
	OutputJSONL
	// OutputJSON 显式 --json：单项保持命令既有形状，多项输出 JSON 数组。
	OutputJSON
)

// ResolveOutputMode 校验互斥并解析输出模式；未显式指定时按 TTY 状态默认。
func ResolveOutputMode(flagJSONL, flagText, flagJSON bool, isTerminal bool) (OutputMode, error) {
	explicit := 0
	for _, set := range []bool{flagJSONL, flagText, flagJSON} {
		if set {
			explicit++
		}
	}
	if explicit > 1 {
		return OutputAuto, errors.New("--jsonl, --text and --json are mutually exclusive")
	}
	switch {
	case flagJSONL:
		return OutputJSONL, nil
	case flagText:
		return OutputText, nil
	case flagJSON:
		return OutputJSON, nil
	case isTerminal:
		return OutputText, nil
	default:
		return OutputJSONL, nil
	}
}

// Writer 按输出模式逐条写出信封；OutputJSON 模式在 Finish 时按 cardinality
// 输出（单信封对象，多信封数组）。
type Writer struct {
	mode     OutputMode
	out      io.Writer
	jsonl    *json.Encoder
	buffered []Envelope
}

// NewWriter 构造按 mode 输出的 writer。
func NewWriter(out io.Writer, mode OutputMode) *Writer {
	writer := &Writer{mode: mode, out: out}
	if mode == OutputJSONL {
		writer.jsonl = json.NewEncoder(out)
	}
	return writer
}

// Mode 返回解析后的输出模式。
func (w *Writer) Mode() OutputMode {
	if w == nil {
		return OutputAuto
	}
	return w.mode
}

// Write 写出一个信封；OutputJSON 模式先缓冲。
func (w *Writer) Write(envelope Envelope) error {
	if w == nil {
		return nil
	}
	switch w.mode {
	case OutputJSONL:
		return w.jsonl.Encode(envelope)
	case OutputText:
		ref := envelope.Ref
		if ref == "" {
			ref = envelope.ID
		}
		if ref == "" {
			return fmt.Errorf("envelope has no printable ref or id")
		}
		_, err := fmt.Fprintln(w.out, ref)
		return err
	default:
		w.buffered = append(w.buffered, envelope)
		return nil
	}
}

// Finish 结束输出：OutputJSON 模式按 cardinality 写出对象或数组；其余模式
// 无操作。
func (w *Writer) Finish() error {
	if w == nil || w.mode != OutputJSON {
		return nil
	}
	var value any
	switch len(w.buffered) {
	case 0:
		value = []Envelope{}
	case 1:
		value = w.buffered[0]
	default:
		value = w.buffered
	}
	line, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.out.Write(line)
	return err
}

// Count 返回已写入/缓冲的信封数量。
func (w *Writer) Count() int {
	if w == nil {
		return 0
	}
	return len(w.buffered)
}
