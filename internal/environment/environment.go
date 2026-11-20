// Package environment owns CLI streams and terminal state.
package environment

import (
	"io"
	"log/slog"

	"github.com/FlanChanXwO/javdb-cli/internal/logging"
)

// Streams 保存调用方提供的标准输入、标准输出和错误输出。
type Streams struct {
	Logger  *slog.Logger
	Command string
	In      io.Reader
	Out     io.Writer
	Err     io.Writer

	// InIsTerminal 标记 stdin 是否为交互终端；由根命令入口探测，测试可显式设置。
	// 非 TTY stdin 是管道/批处理输入的信号，命令据此选择输入分类。
	InIsTerminal bool

	// OutIsTerminal 标记 stdout 是否为交互终端；由根命令入口探测，测试可显式
	// 设置。非 TTY stdout 时默认输出稳定记录流（一行一个 ref/URI），TTY 时
	// 使用人类文本渲染。显式 --json/--ndjson 不受此标记影响。
	OutIsTerminal bool
}

// NewStreams 构造 Streams。
func NewStreams(in io.Reader, out, err io.Writer) *Streams {
	return &Streams{In: in, Out: out, Err: err}
}

// Failure reports using the current stderr, including callers that replace streams after construction.
func (s *Streams) Failure(err error, attrs ...slog.Attr) error {
	logger := s.Logger
	if logger == nil {
		var createErr error
		logger, createErr = logging.New(s.Err, "info", "text")
		if createErr != nil {
			return createErr
		}
	}
	return logging.Failure(logger, err, attrs...)
}

// Diagnostics keeps stderr notes in the selected log format while preserving Err's identity.
func (s *Streams) Diagnostics(level slog.Level) io.Writer {
	logger := s.Logger
	if logger == nil {
		var err error
		logger, err = logging.New(s.Err, "info", "text")
		if err != nil {
			panic(err)
		}
	}
	return logging.Writer(logger, level)
}
