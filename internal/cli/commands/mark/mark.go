// Package mark 提供影片标记命令。
package mark

import (
	"bufio"
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the mark command (watched or want).
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var watched, want, isID bool
	var asJSON, asJSONL, asText bool
	var score int
	var content string
	fetch := func(c *javdb.Client, ctx context.Context, ref string, useID bool, status string) (string, map[string]any, error) {
		mid := ref
		var err error
		if !useID {
			mid, err = c.ResolveMovieID(ctx, ref)
			if err != nil {
				return "", nil, err
			}
		}
		rev, err := c.Mark(ctx, mid, status, score, content)
		if err != nil {
			return "", nil, fmt.Errorf("mark failed: %w", err)
		}
		return mid, rev, nil
	}
	runner := &pipeline.BatchRunner{
		Name:  "mark",
		Kinds: []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			ref := pipeline.ConsumerRef(input)
			status := "want_watch"
			if watched {
				status = "watched"
			}
			mid, rev, err := fetch(c, ctx, ref, isID && input.ID == "", status)
			if err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindMovie, input.Ref, mid).
				WithData(map[string]any{"movie_id": mid, "status": status, "review_id": display(rev["id"])}), nil
		},
		Legacy: func(args []string) error {
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				status := "want_watch"
				label := "想看"
				if watched {
					status = "watched"
					label = "看過"
				}
				mid, rev, err := fetch(c, context.Background(), args[0], isID, status)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(streams.Out, "已标记 %s (%s) → %s\treview_id=%s\n",
					args[0], mid, label, display(rev["id"]))
				return err
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "mark NUMBER",
		Short: "Mark a movie as 看過 (--watched) or 想看 (--want)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if watched == want && (len(args) == 1 || stdinHasContent(streams)) {
				return fmt.Errorf("specify exactly one of --watched or --want")
			}
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&watched, "watched", false, "Mark as 看過")
	cmd.Flags().BoolVar(&want, "want", false, "Mark as 想看")
	cmd.Flags().IntVar(&score, "score", 0, "Optional score")
	cmd.Flags().StringVar(&content, "content", "", "Optional review text")
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

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

// stdinHasContent 报告非 TTY stdin 是否有内容（不消费）。
func stdinHasContent(streams *invocation.Streams) bool {
	if streams.InIsTerminal {
		return false
	}
	reader := bufio.NewReader(streams.In)
	_, err := reader.Peek(1)
	return err == nil
}
