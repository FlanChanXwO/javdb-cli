package record

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// ListProducer 是"拉取列表"类 producer 命令的通用执行器：
// 不消费 stdin；默认输出人类文本；显式 --ndjson 输出逐条信封，--json 走 JSON 载荷。
type ListProducer struct {
	Name string
	// ClientFactory 每次执行调用一次。
	ClientFactory func() (*session.Client, error)
	// Fetch 执行并返回列表项。
	Fetch func(context.Context, *session.Client) ([]map[string]any, error)
	// JSON 构造显式 --json 的既有 shape 载荷。
	JSON func([]map[string]any) (map[string]any, error)
	// ErrNote 可选：输出前写 stderr 注解。
	ErrNote func(io.Writer, []map[string]any)
	// ItemKind 是信封 kind（默认 movie）。
	ItemKind Kind
	// ItemRef 提取 (ref, id)（默认 number/id）。
	ItemRef func(map[string]any) (string, string)
	// RowText 渲染人类文本行（默认 movie 投影行）。
	RowText func(io.Writer, io.Writer, []map[string]any) error
}

// Execute 是 producer 命令 RunE 的通用实现。
func (p *ListProducer) Execute(streams *environment.Streams, ndjson, json bool) error {
	mode, err := ResolveOutputMode(ndjson, json, streams.OutIsTerminal)
	if err != nil {
		return err
	}
	c, err := p.ClientFactory()
	if err != nil {
		return err
	}
	items, err := p.Fetch(context.Background(), c)
	if err != nil {
		return err
	}
	switch mode {
	case OutputJSON:
		payload, err := p.JSON(items)
		if err != nil {
			return err
		}
		return jsonxWrite(streams.Out, payload)
	case OutputText:
		if p.ErrNote != nil {
			p.ErrNote(streams.Diagnostics(slog.LevelInfo), items)
		}
		itemRef := p.ItemRef
		if itemRef == nil {
			itemRef = movieItemRef
		}
		for _, item := range items {
			ref, id := itemRef(item)
			if ref == "" {
				ref = id
			}
			if ref == "" {
				return fmt.Errorf(i18n.T("record.item_missing_ref"), p.Name)
			}
			if _, err := fmt.Fprintln(streams.Out, ref); err != nil {
				return err
			}
		}
		return nil
	case OutputHuman:
		if p.ErrNote != nil {
			p.ErrNote(streams.Diagnostics(slog.LevelInfo), items)
		}
		rowText := p.RowText
		if rowText == nil {
			rowText = WriteMovieRowsText
		}
		return rowText(streams.Out, streams.Diagnostics(slog.LevelInfo), items)
	default:
		if p.ErrNote != nil {
			p.ErrNote(streams.Diagnostics(slog.LevelInfo), items)
		}
		kind := p.ItemKind
		if kind == "" {
			kind = KindMovie
		}
		itemRef := p.ItemRef
		if itemRef == nil {
			itemRef = movieItemRef
		}
		writer := NewWriter(streams.Out, OutputNDJSON)
		for _, item := range items {
			ref, id := itemRef(item)
			envelope := New(kind, ref, id).WithData(map[string]any{"item": item})
			if err := writer.Write(envelope); err != nil {
				return err
			}
		}
		return nil
	}
}

func movieItemRef(item map[string]any) (string, string) {
	return fmt.Sprint(item["number"]), fmt.Sprint(item["id"])
}

// MovieListProducer 是影片列表 producer 的便捷别名。
type MovieListProducer = ListProducer

// WriteMovieRowsText 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func WriteMovieRowsText(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range display.ProjectMovies(movies) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}

func jsonxWrite(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
