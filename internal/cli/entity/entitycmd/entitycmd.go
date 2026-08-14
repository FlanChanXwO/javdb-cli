// Package entitycmd 提供六个实体命令的共享管道化实现。
package entitycmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New 构造实体影片列表命令（actor/series/maker/director/code/list 共用）。
func New(kind, use, short string, options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var (
		zone, sort, order string
		page, limit       int
		tagRefs, main     []string
		allPages          bool
		hasMagnets        bool
		asJSON, asNDJSON  bool
	)
	pipelineKind := pipeline.Kind(kind)
	runner := &pipeline.BatchRunner{
		Name:       kind,
		LegacyJSON: true,
		Kinds:      []pipeline.Kind{pipelineKind},
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			ref := pipeline.ConsumerRef(input)
			res, err := entity.Execute(ctx, c, kind, ref, entity.Options{
				Zone: zone, Sort: sort, Order: order, Page: page, Limit: limit,
				TagRefs: tagRefs, Main: main, AllPages: allPages, HasMagnets: hasMagnets,
			})
			if err != nil {
				return pipeline.Envelope{}, err
			}
			envelope := pipeline.New(pipelineKind, ref, res.EntityID).
				WithData(map[string]any{"entity": res.Entity, "entity_id": res.EntityID, "movies": res.Movies})
			return envelope, nil
		},
		Legacy: func(args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			res, err := entity.Execute(context.Background(), c, kind, args[0], entity.Options{
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
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringArrayVar(&tagRefs, "tag", nil, "Content tag id/EN/中文 (repeatable)")
	cmd.Flags().StringArrayVar(&main, "main", nil, "Main flag p|m|c|s|i|v (repeatable)")
	cmd.Flags().StringVar(&sort, "sort", "release", "hit|release|score|update|want_watch_count|watched_count")
	cmd.Flags().StringVar(&order, "order", "desc", "asc|desc")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&allPages, "all", false, "Fetch all pages (capped)")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON with entity meta + movies")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}

// writeMovieRows 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovieRows(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectMovies(movies) {
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
