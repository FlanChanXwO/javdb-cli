// Package command 提供六个实体命令的共享管道化实现。
package command

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/internal/target"
)

// New 构造实体影片列表命令（actor/series/maker/director/code/list 共用）。
func New(kind, use, short string, options *session.Options, streams *environment.Streams) *cobra.Command {
	var (
		zone, sort, order string
		page, limit       int
		tagRefs, main     []string
		allPages          bool
		hasMagnets        bool
		asJSON, asNDJSON  bool
	)
	pipelineKind := record.Kind(kind)
	runner := &record.BatchRunner{
		Name:       kind,
		LegacyJSON: true,
		Kinds:      []record.Kind{pipelineKind},
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		RunOne: func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			ref := record.ConsumerRef(input)
			res, err := target.Execute(ctx, c, kind, ref, target.Options{
				Zone: zone, Sort: sort, Order: order, Page: page, Limit: limit,
				TagRefs: tagRefs, Main: main, AllPages: allPages, HasMagnets: hasMagnets,
			})
			if err != nil {
				return record.Envelope{}, err
			}
			envelope := record.New(pipelineKind, ref, res.EntityID).
				WithData(map[string]any{"entity": res.Entity, "entity_id": res.EntityID, "movies": res.Movies})
			return envelope, nil
		},
		Legacy: func(args []string) error {
			c, err := session.NewWithDefaultToken(options)
			if err != nil {
				return err
			}
			res, err := target.Execute(context.Background(), c, kind, args[0], target.Options{
				Zone: zone, Sort: sort, Order: order, Page: page, Limit: limit,
				TagRefs: tagRefs, Main: main, AllPages: allPages, HasMagnets: hasMagnets,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(streams.Out, map[string]any{
					"entity": res.Entity, "entity_id": res.EntityID, "movies": res.Movies,
				})
			}
			return writeMovieRows(streams.Out, streams.Err, res.Movies)
		},
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", i18n.T("cli.censored_uncensored_western_fc"))
	cmd.Flags().StringArrayVar(&tagRefs, "tag", nil, i18n.T("cli.content_tag_id_en_repeatable"))
	cmd.Flags().StringArrayVar(&main, "main", nil, i18n.T("cli.main_flag_p_m_c_s_i_v_repeatable"))
	cmd.Flags().StringVar(&sort, "sort", "release", i18n.T("cli.hit_release_score_update_want_watch_count_watched_count"))
	cmd.Flags().StringVar(&order, "order", "desc", i18n.T("cli.asc_desc"))
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("cli.page"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("cli.page_size"))
	cmd.Flags().BoolVar(&allPages, "all", false, i18n.T("cli.fetch_all_pages_capped"))
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("cli.drop_magnets_count"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.json_with_entity_meta_movies"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}

// writeMovieRows 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovieRows(w, errW io.Writer, movies []map[string]any) error {
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

// writeJSON 以 jsonx.MarshalLine 写出紧凑 JSON 并传播编码与写入错误。
func writeJSON(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
