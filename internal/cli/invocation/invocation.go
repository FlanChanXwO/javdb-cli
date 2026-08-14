// Package invocation 保存 CLI 根命令的调用期数据：persistent flags 目标与标准流。
//
// 只保存调用方提供的值，不包装、缓存或替换 IO，不含服务方法；不导入 config、auth、
// SDK、Cobra 或 update。
package invocation

import "io"

// RootOptions 是根命令 persistent flags 的目标。
type RootOptions struct {
	Proxy string
	Host  string
}

// Streams 保存调用方提供的标准输入、标准输出和错误输出。
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer

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
