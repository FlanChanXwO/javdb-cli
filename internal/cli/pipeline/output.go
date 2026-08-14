package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// OutputMode 是命令的输出格式。
type OutputMode int

const (
	// OutputAuto 表示尚未解析显式输出 flag。
	OutputAuto OutputMode = iota
	// OutputHuman 是 TTY 默认：人类可读文本，由命令的 RenderText 渲染。
	OutputHuman
	// OutputText 是非 TTY 默认：稳定记录流，一行一个 ref/URI，便于管道消费。
	OutputText
	// OutputNDJSON 强制逐条信封 NDJSON。
	OutputNDJSON
	// OutputJSON 显式 --json：单项保持命令既有形状，多项输出 JSON 数组。
	OutputJSON
)

// ResolveOutputMode 校验互斥并解析输出模式：显式 --ndjson/--json 优先且互斥；
// 未显式指定时，TTY stdout 使用人类文本（OutputHuman），非 TTY stdout 使用
// 稳定记录流（OutputText）。测试可显式设置 OutIsTerminal 来模拟两种场景。
func ResolveOutputMode(flagNDJSON, flagJSON bool, outIsTerminal bool) (OutputMode, error) {
	if flagNDJSON && flagJSON {
		return OutputAuto, errors.New("--ndjson and --json are mutually exclusive")
	}
	switch {
	case flagNDJSON:
		return OutputNDJSON, nil
	case flagJSON:
		return OutputJSON, nil
	case outIsTerminal:
		return OutputHuman, nil
	default:
		return OutputText, nil
	}
}

// Writer 按输出模式逐条写出信封；OutputJSON 模式在 Finish 时按 cardinality
// 输出（单信封对象，多信封数组）。
type Writer struct {
	mode     OutputMode
	out      io.Writer
	ndjson   *json.Encoder
	buffered []Envelope
}

// NewWriter 构造按 mode 输出的 writer。
func NewWriter(out io.Writer, mode OutputMode) *Writer {
	writer := &Writer{mode: mode, out: out}
	if mode == OutputNDJSON {
		writer.ndjson = json.NewEncoder(out)
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

// Write 写出一个信封；OutputJSON 模式先缓冲。OutputHuman 在 Writer 层面
// 退化为 OutputText（逐行 ref）；命令级别的人类渲染由 Consumer.RenderText
// 或 Producer.RenderText 负责，在进入 Writer 前已完成分流。
func (w *Writer) Write(envelope Envelope) error {
	if w == nil {
		return nil
	}
	switch w.mode {
	case OutputNDJSON:
		return w.ndjson.Encode(envelope)
	case OutputText, OutputHuman:
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
