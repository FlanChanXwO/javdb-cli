// Package lists 提供我的合集与公开合集命令组。
package lists

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// New builds the personal and public list command tree.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var page, limit int
	var sortBy string
	var asJSON bool
	producer := &pipeline.Producer{
		Name: "lists",
		Produce: func(ctx context.Context) ([]pipeline.Envelope, error) {
			c, err := client.NewWithDefaultToken(options)
			if err != nil {
				return nil, err
			}
			res, err := c.MyLists(ctx, page, limit, sortBy)
			if err != nil {
				return nil, fmt.Errorf("lists failed: %w", err)
			}
			items := res.Named("lists")
			envelopes := make([]pipeline.Envelope, 0, len(items))
			for _, item := range items {
				envelopes = append(envelopes, pipeline.New(pipeline.KindList, display(item["name"]), display(item["id"])).WithData(map[string]any{"list": item}))
			}
			return envelopes, nil
		},
		RenderText: func(w io.Writer, envelopes []pipeline.Envelope) error {
			items := make([]map[string]any, 0, len(envelopes))
			for _, envelope := range envelopes {
				if list, ok := envelope.Data["list"].(map[string]any); ok {
					items = append(items, list)
				}
			}
			return writeListRows(w, streams.Err, items)
		},
		LegacyJSON: func(w io.Writer) error {
			c, err := client.NewWithDefaultToken(options)
			if err != nil {
				return err
			}
			res, err := c.MyLists(context.Background(), page, limit, sortBy)
			if err != nil {
				return fmt.Errorf("lists failed: %w", err)
			}
			items := res.Named("lists")
			return writeJSON(w, map[string]any{
				"lists":        items,
				"current_page": jsonx.RawString(res["current_page"]),
			})
		},
	}
	var asNDJSON bool
	cmd := &cobra.Command{
		Use:   "lists",
		Short: "My 合集; subcommands: show/search/related",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().StringVar(&sortBy, "sort-by", "created", "created|name|movies_count|views_count|updated|default")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")

	cmd.AddCommand(NewShow(options, streams))
	cmd.AddCommand(NewSearch(options, streams))
	cmd.AddCommand(NewRelated(options, streams))
	return cmd
}

// writeListRows 写出 id\tname\tmovies\tprivacy\tviews 行；空列表输出 (空列表)。
func writeListRows(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			display(item["id"]), display(item["name"]), display(item["movies_count"]),
			display(item["privacy"]), display(item["views_count"])); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON 以 jsonx.MarshalLine 写出紧凑 JSON 并传播编码与写入错误。
func writeJSON(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
