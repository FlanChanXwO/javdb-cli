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
	// Context 是本次命令调用的生命周期；未设置时使用 Background。
	Context context.Context
	// Kinds 是文本 ref 输入分配的 kind 与消费者接受的 kind。
	Kinds []Kind
	// ClientFactory 每次批量执行调用一次，返回携带默认 token 的 client。
	ClientFactory func() (*javdb.Client, error)
	// RunOne 执行单项并返回输出信封。
	RunOne func(*javdb.Client, context.Context, Envelope) (Envelope, error)
	// RunMany 执行单项并返回多个输出信封（fan-out）；设置时优先于 RunOne。
	// 调用方负责在返回的信封中携带稳定 ref/id。
	RunMany func(*javdb.Client, context.Context, Envelope) ([]Envelope, error)
	// Legacy 处理单项既有路径（args 为位置参数列表）。
	Legacy func(args []string) error
	// RenderText 是 pipeline 文本/人类模式的领域投影；nil 时输出稳定 ref。
	RenderText func(io.Writer, Envelope) error
	// RenderError 是 pipeline 文本/人类模式的错误投影；nil 使用 Name 前缀。
	RenderError func(io.Writer, error) error
	// RouteTextThroughPipeline 让单项纯文本输入的非 TTY OutputText 走 Consumer。
	// 未启用时保留本地/有副作用命令的既有 Legacy 文本语义；需要稳定记录的
	// 只读命令应显式启用此选项。
	RouteTextThroughPipeline bool
	// LegacyJSON 表示 Legacy 路径已支持显式 --json 输出（保持既有 shape）。
	// 为 false 时，单项显式 --json 也走 consumer 路径（RunOne + 信封对象输出），
	// 避免本地命令（mark/config 等）在 --json 下静默输出裸文本或无输出。
	LegacyJSON bool
	// Preflight 在批处理路径开始前校验全部输入（如 download 的全量目标展开
	// 与冲突检查）；返回错误时整个批处理失败，不做任何写入。
	Preflight func([]Envelope) error
	// Concurrency 控制批量执行并发度；<= 0 表示串行。
	Concurrency int
}

// Execute 是命令 RunE 的通用实现。
func (b *BatchRunner) Execute(streams *invocation.Streams, args []string, ndjson, json bool) error {
	mode, err := ResolveOutputMode(ndjson, json, streams.OutIsTerminal)
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
	// 单项 TTY 人类文本或显式 --json（且 Legacy 支持 JSON）
	// 且输入是纯文本 ref（无 kind）时走既有路径，保持 shape 与认证语义；
	// 启用 RouteTextThroughPipeline 的只读命令在非 TTY OutputText 下经过
	// pipeline，输出稳定、可消费的记录；其他命令保留既有 Legacy 文本语义；
	// NDJSON 信封即使只有一条也必须经过 kind 校验并保留 id 语义，绝不绕过
	// 消费者检查。Legacy 不支持 JSON 的命令在显式 --json 下走 consumer 路径
	// 输出信封对象。
	if len(inputs) == 1 && inputs[0].Kind == "" && (mode == OutputHuman || (mode == OutputText && !b.RouteTextThroughPipeline) || (mode == OutputJSON && b.LegacyJSON)) {
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
		Context:       b.Context,
		AcceptedKinds: b.Kinds,
		Concurrency:   b.Concurrency,
		RenderText:    b.RenderText,
		RenderError:   b.RenderError,
		RunOne: func(ctx context.Context, input Envelope) (Envelope, error) {
			return b.RunOne(client, ctx, input)
		},
	}
	if b.RunMany != nil {
		consumer.RunMany = func(ctx context.Context, input Envelope) ([]Envelope, error) {
			return b.RunMany(client, ctx, input)
		}
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
	mode, err := ResolveOutputMode(ndjson, json, streams.OutIsTerminal)
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
		for _, envelope := range envelopes {
			if err := writeRef(streams.Out, envelope); err != nil {
				return err
			}
		}
		return nil
	case OutputHuman:
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
