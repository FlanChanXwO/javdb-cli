// Package lists 提供我的合集与公开合集命令组。
package lists

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the personal and public list command tree.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var page, limit int
	var sortBy string
	var asJSON bool
	var currentPage string
	producer := &record.Producer{
		Name: "lists",
		Produce: func(ctx context.Context) ([]record.Envelope, error) {
			c, err := session.NewRequired(options)
			if err != nil {
				return nil, err
			}
			res, err := c.MyLists(ctx, page, limit, sortBy)
			if err != nil {
				return nil, err
			}
			currentPage = jsonx.RawString(res["current_page"])
			items := res.Named("lists")
			envelopes := make([]record.Envelope, 0, len(items))
			for _, item := range items {
				envelope, err := listEnvelope(item)
				if err != nil {
					return nil, err
				}
				envelopes = append(envelopes, envelope)
			}
			return envelopes, nil
		},
		RenderText: func(w io.Writer, envelopes []record.Envelope) error {
			items := make([]map[string]any, 0, len(envelopes))
			for _, envelope := range envelopes {
				if list, ok := envelope.Data["list"].(map[string]any); ok {
					items = append(items, list)
				}
			}
			return writeListRows(w, streams.Err, items)
		},
		RenderJSON: func(w io.Writer, envelopes []record.Envelope) error {
			items := make([]map[string]any, 0, len(envelopes))
			for _, envelope := range envelopes {
				list, ok := envelope.Data["list"].(map[string]any)
				if !ok {
					return i18n.Errorf("lists.missing_data")
				}
				items = append(items, list)
			}
			return writeJSON(w, map[string]any{
				"lists":        items,
				"current_page": currentPage,
			})
		},
	}
	var asNDJSON bool
	cmd := &cobra.Command{
		Use:   "lists",
		Short: i18n.T("command.lists"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("flag.page"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("flag.limit"))
	cmd.Flags().StringVar(&sortBy, "sort-by", "created", "created|name|movies_count|views_count|updated|default")
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))

	cmd.AddCommand(NewShow(options, streams))
	cmd.AddCommand(NewSearch(options, streams))
	cmd.AddCommand(NewRelated(options, streams))
	return cmd
}

// writeListRows 写出 id\tname\tmovies\tprivacy\tviews 行；空列表输出 (空列表)。
func writeListRows(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(errW, i18n.T("result.empty"))
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", display(item["id"]), display(item["name"]), display(item["movies_count"]),
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
