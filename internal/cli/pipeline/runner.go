package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// BatchRunner 是只读命令的管道化执行器：
//   - 单项 + （TTY 文本 或 显式 --json）走 Legacy 既有路径（保持既有 shape
//     与可选认证/匿名重试行为）。
//   - 其余情况（多项，或单项显式 --jsonl）走逐项 RunOne：单项失败原位错误
//     信封并继续，最终非零；批量显式 --json 输出信封数组。
type BatchRunner struct {
	Name string
	// Kinds 是文本 ref 输入分配的 kind 与消费者接受的 kind。
	Kinds []Kind
	// ClientFactory 每次批量执行调用一次，返回携带默认 token 的 client。
	ClientFactory func() (*javdb.Client, error)
	// RunOne 执行单项并返回输出信封。
	RunOne func(*javdb.Client, context.Context, Envelope) (Envelope, error)
	// Legacy 处理单项既有路径（args 为位置参数列表）。
	Legacy func(args []string) error
}

// Execute 是命令 RunE 的通用实现。
func (b *BatchRunner) Execute(streams *invocation.Streams, args []string, jsonl, text, json bool) error {
	mode, err := ResolveOutputMode(jsonl, text, json, streams.InIsTerminal)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(streams.In)
	arg := ""
	if len(args) == 1 {
		arg = args[0]
	}
	inputs, err := CollectInputs(reader, streams, arg, b.Kinds)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return fmt.Errorf("keyword or an image")
	}
	return b.ExecuteWithInputs(streams, inputs, mode)
}

// ExecuteWithInputs 处理已收集的输入（调用方已完成分类）。
func (b *BatchRunner) ExecuteWithInputs(streams *invocation.Streams, inputs []Envelope, mode OutputMode) error {
	if len(inputs) == 1 && (mode == OutputText || mode == OutputJSON) {
		// 单项 TTY 文本或显式 --json：既有路径，保持 shape 与认证语义。
		return b.Legacy([]string{pipelineConsumerRef(inputs[0])})
	}
	client, err := b.ClientFactory()
	if err != nil {
		return err
	}
	consumer := &Consumer{
		Name:          b.Name,
		AcceptedKinds: b.Kinds,
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			return b.RunOne(client, ctx, input)
		},
	}
	return consumer.RunInputs(streams, inputs, mode)
}

func pipelineConsumerRef(input Envelope) string {
	if input.ID != "" {
		return input.ID
	}
	return input.Ref
}

// Producer 是无位置参数命令的非 TTY 输出器：不消费 stdin，非 TTY 时把结果
// 逐条输出为信封；TTY 走 Text 渲染；显式 --json 走 LegacyJSON。
type Producer struct {
	Name string
	// Produce 执行并返回输出信封序列（空切片表示无结果）。
	Produce func(context.Context) ([]Envelope, error)
	// RenderText 渲染人类文本。
	RenderText func(io.Writer, []Envelope) error
	// LegacyJSON 输出显式 --json 的既有 shape。
	LegacyJSON func(io.Writer) error
}

// Execute 是 producer 命令 RunE 的通用实现。
func (p *Producer) Execute(streams *invocation.Streams, jsonl, text, json bool) error {
	mode, err := ResolveOutputMode(jsonl, text, json, streams.InIsTerminal)
	if err != nil {
		return err
	}
	envelopes, err := p.Produce(context.Background())
	if err != nil {
		return err
	}
	switch mode {
	case OutputJSON:
		return p.LegacyJSON(streams.Out)
	case OutputText:
		return p.RenderText(streams.Out, envelopes)
	default:
		writer := NewWriter(streams.Out, OutputJSONL)
		for _, envelope := range envelopes {
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
		return nil
	}
}
