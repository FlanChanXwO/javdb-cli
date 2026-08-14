package pipeline

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// KindTag 是标签目录条目的稳定 kind（kind 集合的最小契约之外按需扩展）。
const KindTag Kind = "tag"

// Consumer 把只读命令的单项执行包装成统一输入/输出管道：
//   - 输入：位置参数或非 TTY stdin（分类：图片 magic → NDJSON → 文本）。
//   - 消费者严格检查 kind，不兼容输入生成原位错误信封。
//   - 输出：默认文本；显式 --ndjson/--json 互斥。
//   - 显式 --json：单项走 LegacyJSON 既有 shape，多项输出信封数组。
type Consumer struct {
	Name string
	// AcceptedKinds 是输入项允许的 kind；nil 表示接受任意 kind（文本 ref）。
	AcceptedKinds []Kind
	// RunOne 执行单项并返回输出信封。
	RunOne func(context.Context, Envelope) (Envelope, error)
	// RenderText 输出默认的人类文本。
	RenderText func(io.Writer, Envelope) error
	// LegacyJSON 输出单项显式 --json 的既有 shape；nil 时输出信封对象。
	LegacyJSON func(io.Writer, Envelope) error
}

// Execute 是命令 RunE 的通用实现（不消费图片；需要图片输入的调用方自行先行处理）。
func (c *Consumer) Execute(streams *invocation.Streams, args []string, ndjson, json bool) error {
	mode, err := ResolveOutputMode(ndjson, json, streams.OutIsTerminal)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(streams.In)
	arg := ""
	if len(args) == 1 {
		arg = args[0]
	}
	inputs, err := CollectInputs(reader, streams, arg, c.AcceptedKinds)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return fmt.Errorf("keyword or an image")
	}
	return c.RunInputs(streams, inputs, mode)
}

// RunInputs 按输出模式处理已收集的输入；单项失败原位错误信封并继续。
// 调用方（如 search）可自行完成图片分类后复用本方法。
//
// 文本/人类模式（OutputText/OutputHuman）下：
//   - 成功项：若 RenderText 已设置则调用它渲染人类/稳定文本，否则逐行 ref。
//   - 失败项：错误原因写 stderr，不向 stdout 写任何内容（不伪造成功 ref）。
//
// NDJSON/JSON 模式下：失败项原位 error 信封写入 stdout（机器契约），保持
// 输出顺序与单项错误可观测性。
func (c *Consumer) RunInputs(streams *invocation.Streams, inputs []Envelope, mode OutputMode) error {
	// legacy JSON shape 只适用于纯文本 ref 输入；NDJSON 信封先过 kind 校验。
	if mode == OutputJSON && len(inputs) == 1 && inputs[0].Kind == "" && c.LegacyJSON != nil {
		output, err := c.RunOne(context.Background(), inputs[0])
		if err != nil {
			if legacyErr := c.LegacyJSON(streams.Out, ErrorEnvelope(inputs[0], c.Name, "batch", "item", err.Error())); legacyErr != nil {
				return legacyErr
			}
			return err
		}
		return c.LegacyJSON(streams.Out, output)
	}

	// 文本/人类模式：成功项按 RenderText 或 ref 写 stdout，失败项写 stderr。
	if mode == OutputText || mode == OutputHuman {
		failures := 0
		for _, input := range inputs {
			if !c.accepts(input) {
				failures++
				fmt.Fprintf(streams.Err, "%s: %s\n", c.Name, fmt.Sprintf("unsupported kind %q", input.Kind))
				continue
			}
			output, err := c.RunOne(context.Background(), input)
			if err != nil {
				failures++
				fmt.Fprintf(streams.Err, "%s: %s\n", c.Name, err.Error())
				continue
			}
			if c.RenderText != nil {
				if err := c.RenderText(streams.Out, output); err != nil {
					return err
				}
			} else if err := writeRef(streams.Out, output); err != nil {
				return err
			}
		}
		if failures > 0 {
			return fmt.Errorf("%s completed with %d of %d items failed", c.Name, failures, len(inputs))
		}
		return nil
	}

	writer := NewWriter(streams.Out, mode)
	failures := 0
	for _, input := range inputs {
		if !c.accepts(input) {
			failures++
			output := ErrorEnvelope(input, c.Name, "input", "kind", fmt.Sprintf("unsupported kind %q", input.Kind))
			if err := writer.Write(output); err != nil {
				return err
			}
			continue
		}
		output, err := c.RunOne(context.Background(), input)
		if err != nil {
			failures++
			output = ErrorEnvelope(input, c.Name, "batch", "item", err.Error())
		}
		if err := writer.Write(output); err != nil {
			return err
		}
	}
	if err := writer.Finish(); err != nil {
		return err
	}
	if failures > 0 {
		return fmt.Errorf("%s completed with %d of %d items failed", c.Name, failures, len(inputs))
	}
	return nil
}

// writeRef 写出信封的稳定引用（优先 ref，否则 id），不含错误信封。
func writeRef(w io.Writer, envelope Envelope) error {
	ref := envelope.Ref
	if ref == "" {
		ref = envelope.ID
	}
	if ref == "" {
		return fmt.Errorf("envelope has no printable ref or id")
	}
	_, err := fmt.Fprintln(w, ref)
	return err
}

// CollectInputs 统一收集输入：位置参数（单个）或非 TTY stdin 批。
// 位置参数与非空 stdin 同时存在是歧义错误。stdin 按固定顺序分类
// （图片 magic → NDJSON → 文本）；图片输入显式报错（调用方先行处理）。
func CollectInputs(reader *bufio.Reader, streams *invocation.Streams, arg string, textKinds []Kind) ([]Envelope, error) {
	stdinHasContent := false
	if !streams.InIsTerminal {
		if _, err := reader.Peek(1); err == nil {
			stdinHasContent = true
		} else if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
	}
	if arg != "" && stdinHasContent {
		return nil, fmt.Errorf("ambiguous input: provide either a positional argument or stdin, not both")
	}
	if arg != "" {
		return []Envelope{New("", arg, "")}, nil
	}
	if streams.InIsTerminal || !stdinHasContent {
		return nil, nil
	}
	classification, content, err := Classify(reader)
	if err != nil {
		return nil, err
	}
	switch classification {
	case ClassificationImage:
		return nil, fmt.Errorf("stdin is an image; this command does not accept image input")
	case ClassificationNDJSON:
		return ParseBatch(content, KindNDJSONInput)
	default:
		kind := KindMovie
		if len(textKinds) == 1 {
			kind = textKinds[0]
		}
		return ParseBatch(content, kind)
	}
}

func (c *Consumer) accepts(input Envelope) bool {
	if input.Kind == "" {
		return true
	}
	if len(c.AcceptedKinds) == 0 {
		return true
	}
	for _, kind := range c.AcceptedKinds {
		if input.Kind == kind {
			return true
		}
	}
	return false
}

// ConsumerRef 返回输入项的稳定引用：优先内部 id，否则 ref。
func ConsumerRef(input Envelope) string {
	if input.ID != "" {
		return input.ID
	}
	return input.Ref
}
