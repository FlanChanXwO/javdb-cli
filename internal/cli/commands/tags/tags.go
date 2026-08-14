// Package tags 提供内容标签 taxonomy 命令。
package tags

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

// New builds the taxonomy cache command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var zone string
	var refresh bool
	var asJSON, asNDJSON bool
	producer := &pipeline.Producer{
		Name: "tags",
		Produce: func(ctx context.Context) ([]pipeline.Envelope, error) {
			c, err := client.New(options, "")
			if err != nil {
				return nil, err
			}
			doc, path, err := c.LoadOrRefreshTaxonomy(ctx, zone, refresh)
			if err != nil {
				return nil, fmt.Errorf("tags failed: %w", err)
			}
			if refresh {
				fmt.Fprintf(streams.Err, "taxonomy 已写入 %s\n", path)
			}
			var envelopes []pipeline.Envelope
			if doc == nil {
				return envelopes, nil
			}
			for _, cat := range doc.Categories {
				cname := cat.NameEN
				if cname == "" {
					cname = cat.NameZH
				}
				for _, t := range cat.Tags {
					envelopes = append(envelopes, pipeline.New(pipeline.KindTag, t.NameEN, t.ID).
						WithData(map[string]any{"category_id": cat.ID, "category": cname, "name_zh": t.NameZH}))
				}
			}
			return envelopes, nil
		},
		RenderText: func(w io.Writer, envelopes []pipeline.Envelope) error {
			if len(envelopes) == 0 {
				_, err := fmt.Fprintln(streams.Err, "(空列表)")
				return err
			}
			currentCategory := ""
			for _, envelope := range envelopes {
				category := fmt.Sprint(envelope.Data["category_id"])
				if category != currentCategory {
					currentCategory = category
					fmt.Fprintf(w, "# %s\t%s\n", category, envelope.Data["category"])
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", envelope.ID, envelope.Ref, envelope.Data["name_zh"])
			}
			return nil
		},
		LegacyJSON: func(w io.Writer) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			doc, _, err := c.LoadOrRefreshTaxonomy(context.Background(), zone, false)
			if err != nil {
				return fmt.Errorf("tags failed: %w", err)
			}
			var categories []map[string]any
			if doc != nil {
				for _, cat := range doc.Categories {
					var tagList []map[string]any
					for _, t := range cat.Tags {
						tagList = append(tagList, map[string]any{"id": t.ID, "name_en": t.NameEN, "name_zh": t.NameZH})
					}
					categories = append(categories, map[string]any{"id": cat.ID, "name_en": cat.NameEN, "name_zh": cat.NameZH, "tags": tagList})
				}
			}
			b, err := jsonx.MarshalLine(map[string]any{"categories": categories})
			if err != nil {
				return err
			}
			_, err = w.Write(b)
			return err
		},
	}
	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List content-tag taxonomy (id + EN + 中文)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Re-fetch from API and rewrite local JSON")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}
