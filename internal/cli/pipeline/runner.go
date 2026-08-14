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
//   - 其余情况（多项，或单项显式 --ndjson）走逐项 RunOne：单项失败原位错误
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
	// LegacyJSON 表示 Legacy 路径已支持显式 --json 输出（保持既有 shape）。
	// 为 false 时，单项显式 --json 也走 consumer 路径（RunOne + 信封对象输出），
	// 避免本地命令（mark/config 等）在 --json 下静默输出裸文本或无输出。
	LegacyJSON bool
	// Preflight 在批处理路径开始前校验全部输入（如 download 的全量目标展开
	// 与冲突检查）；返回错误时整个批处理失败，不做任何写入。
	Preflight func([]Envelope) error
}

// Execute 是命令 RunE 的通用实现。
func (b *BatchRunner) Execute(streams *invocation.Streams, args []string, ndjson, json bool) error {
	mode, err := ResolveOutputMode(ndjson, json)
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
	// 单项 TTY 文本或显式 --json（且 Legacy 支持 JSON）且输入是纯文本 ref
	// （无 kind）时走既有路径，保持 shape 与认证语义；NDJSON 信封即使只有一条
	// 也必须经过 kind 校验并保留 id 语义，绝不绕过消费者检查。Legacy 不支持
	// JSON 的命令在显式 --json 下走 consumer 路径输出信封对象。
	if len(inputs) == 1 && inputs[0].Kind == "" && (mode == OutputText || (mode == OutputJSON && b.LegacyJSON)) {
		return b.Legacy([]string{pipelineConsumerRef(inputs[0])})
	}
	if b.Preflight != nil {
		if err := b.Preflight(inputs); err != nil {
			return err
		}
	}
	var client *javdb.Client
	if b.ClientFactory != nil {
		var err error
		client, err = b.ClientFactory()
		if err != nil {
			return err
		}
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

// Producer 是无位置参数命令的输出器：不消费 stdin，默认走 Text 渲染；
// 显式 --ndjson 逐条输出信封，--json 走 LegacyJSON。
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
func (p *Producer) Execute(streams *invocation.Streams, ndjson, json bool) error {
	mode, err := ResolveOutputMode(ndjson, json)
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
		writer := NewWriter(streams.Out, OutputNDJSON)
		for _, envelope := range envelopes {
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
		return nil
	}
}
