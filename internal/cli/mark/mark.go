// Package mark 提供影片标记命令。
package mark

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the mark command (watched or want).
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var watched, want, isID bool
	var asJSON, asNDJSON bool
	var score int
	var content string
	fetch := func(c *session.Client, ctx context.Context, ref string, useID bool, status string) (string, map[string]any, error) {
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
			return "", nil, err
		}
		return mid, rev, nil
	}
	runner := &record.BatchRunner{
		Name:  "mark",
		Kinds: []record.Kind{record.KindMovie},
		ClientFactory: func() (*session.Client, error) {
			return session.NewRequired(options)
		},
		RunOne: func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			ref := record.ConsumerRef(input)
			status := "want_watch"
			if watched {
				status = "watched"
			}
			// 管道信封的内部 ID 是权威目标；无 ID 时再由 --id 决定 raw ref 是否直用。
			mid, rev, err := fetch(c, ctx, ref, input.ID != "" || isID, status)
			if err != nil {
				return record.Envelope{}, err
			}
			return record.New(record.KindMovie, input.Ref, mid).
				WithData(map[string]any{"movie_id": mid, "status": status, "review_id": display(rev["id"])}), nil
		},
		Legacy: func(args []string) error {
			return session.WithRequiredAuth(options, streams.Err, func(c *session.Client) error {
				status := "want_watch"
				label := i18n.T("cli.want_label")
				if watched {
					status = "watched"
					label = i18n.T("cli.watched_label")
				}
				mid, rev, err := fetch(c, context.Background(), args[0], isID, status)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(streams.Out, i18n.T("cli.s_s_s_review_id_s"), args[0], mid, label, display(rev["id"]))
				return err
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "mark NUMBER",
		Short: i18n.T("cli.mark_a_movie_as_watched_or_want"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if watched == want {
				return i18n.Errorf("cli.specify_exactly_one_of_watched_or_want")
			}
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&watched, "watched", false, i18n.T("cli.mark_as"))
	cmd.Flags().BoolVar(&want, "want", false, i18n.T("cli.mark_as_2"))
	cmd.Flags().IntVar(&score, "score", 0, i18n.T("cli.optional_score"))
	cmd.Flags().StringVar(&content, "content", "", i18n.T("cli.optional_review_text"))
	cmd.Flags().BoolVarP(&isID, "id", "i", false, i18n.T("cli.treat_number_as_internal_movie_id"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
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
