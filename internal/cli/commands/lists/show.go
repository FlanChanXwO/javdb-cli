package lists

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewShow builds the lists show command.
func NewShow(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var asJSON, asJSONL, asText bool
	runOne := func(c *javdb.Client, ctx context.Context, ref string) (string, map[string]any, error) {
		eid, err := c.ResolveEntity(ctx, "list", ref, "censored")
		if err != nil {
			return "", nil, fmt.Errorf("lists show failed: %w", err)
		}
		data, err := c.ListInfo(ctx, eid)
		if err != nil {
			return "", nil, fmt.Errorf("lists show failed: %w", err)
		}
		return eid, data, nil
	}
	runner := &pipeline.BatchRunner{
		Name:  "lists show",
		Kinds: []pipeline.Kind{pipeline.KindList},
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			eid, data, err := runOne(c, ctx, pipeline.ConsumerRef(input))
			if err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindList, input.Ref, eid).WithData(map[string]any{"list_info": data}), nil
		},
		Legacy: func(args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			ctx := context.Background()
			eid, data, err := runOne(c, ctx, args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(streams.Out, data)
			}
			_ = eid
			meta, _ := data["list"].(map[string]any)
			if meta == nil {
				if m, ok := data["list"]; ok {
					b, _ := json.Marshal(m)
					_ = json.Unmarshal(b, &meta)
				}
			}
			if meta == nil {
				meta = data
			}
			fmt.Fprintf(streams.Out, "id\t%s\n", coalesce(display(meta["id"]), eid))
			fmt.Fprintf(streams.Out, "name\t%s\n", display(meta["name"]))
			if d := display(meta["description"]); d != "" {
				fmt.Fprintf(streams.Out, "desc\t%s\n", d)
			}
			fmt.Fprintf(streams.Out, "movies\t%s\n", display(meta["movies_count"]))
			fmt.Fprintf(streams.Out, "views\t%s\n", display(meta["views_count"]))
			fmt.Fprintf(streams.Out, "collects\t%s\n", display(meta["collections_count"]))
			if s := display(meta["share_info"]); s != "" {
				fmt.Fprintf(streams.Out, "share\t%s\n", s)
			}
			fmt.Fprintf(streams.Out, "is_creator\t%v\n", data["is_creator"])
			fmt.Fprintf(streams.Out, "has_collected\t%v\n", data["has_collected"])
			return nil
		},
	}
	cmd := &cobra.Command{
		Use:   "show REF",
		Short: "Show 合集 meta (movies: use list <id>)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// display 保持 CLI 既有的数值 ID 截断展示约定。
func display(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return scalar.String(t)
	}
}
