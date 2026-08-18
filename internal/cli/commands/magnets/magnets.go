// Package magnets 提供影片磁力列表与最佳磁力选择命令。
package magnets

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the magnet listing and best-magnet selection command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var (
		cnsub, hd, best, isID bool
		asJSON, asNDJSON      bool
		minSize               string
	)
	fetch := func(c *javdb.Client, ctx context.Context, input pipeline.Envelope, useID bool) (string, []map[string]any, error) {
		mid := pipeline.ConsumerRef(input)
		var err error
		if !useID {
			mid, err = c.ResolveMovieID(ctx, mid)
			if err != nil {
				return "", nil, err
			}
		}
		// 若输入信封已携带 magnets_count，跳过重复 MovieDetail 请求。
		count, hasCount := knownMagnetCount(input)
		var items []map[string]any
		if hasCount && count == 0 {
			items = nil
		} else {
			if hasCount {
				// magnets_count > 0：直接取磁力，不做详情请求。
				items, err = c.MovieMagnets(ctx, mid)
			} else {
				detail, derr := c.MovieDetail(ctx, mid)
				if derr != nil {
					return "", nil, fmt.Errorf("magnets failed: %w", derr)
				}
				if magnetCount(detail["magnets_count"]) == 0 {
					items = nil
				} else {
					items, err = c.MovieMagnets(ctx, mid)
				}
			}
			if err != nil {
				return "", nil, fmt.Errorf("magnets failed: %w", err)
			}
		}
		minMiB := 0
		if minSize != "" {
			minMiB, err = ParseSizeMiB(minSize)
			if err != nil {
				return "", nil, err
			}
		}
		items = javdb.FilterMagnets(items, cnsub, hd, minMiB)
		return mid, items, nil
	}
	runOne := func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
		// 管道信封携带 id 时直接按内部 ID 请求，不做番号解析。
		// --id flag 仅对位置参数（无信封 id）生效。
		useID := input.ID != "" || isID
		mid, items, err := fetch(c, ctx, input, useID)
		if err != nil {
			return pipeline.Envelope{}, err
		}
		data := map[string]any{"movie_id": mid, "magnets": items}
		if best {
			b := javdb.PickBestMagnet(items)
			data["best"] = b
			data["magnet_uri"] = javdb.MagnetURI(b)
		}
		return pipeline.New(pipeline.KindMagnet, input.Ref, mid).WithData(data), nil
	}
	runWithOptionalAuth := func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
		var output pipeline.Envelope
		err := client.RetryAnonymousAuth(c, streams.Err, func() error {
			var runErr error
			output, runErr = runOne(c, ctx, input)
			return runErr
		})
		return output, err
	}
	runner := &pipeline.BatchRunner{
		Name:       "magnets",
		LegacyJSON: true,
		// 非 TTY 单项也输出可供下游消费的磁力 URI。
		RouteTextThroughPipeline: true,
		RenderText: func(w io.Writer, envelope pipeline.Envelope) error {
			return renderText(w, envelope, best)
		},
		Kinds: []pipeline.Kind{pipeline.KindMovie},
		// 逐项请求磁力并按输入顺序写出；保持串行以避免放大 API 请求。
		Concurrency: 1,
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			return runWithOptionalAuth(c, ctx, input)
		},
		Legacy: func(args []string) error {
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
				ctx := context.Background()
				mid, items, err := fetch(c, ctx, pipeline.New("", args[0], ""), isID)
				if err != nil {
					return err
				}
				if best {
					b := javdb.PickBestMagnet(items)
					if asJSON {
						return writeJSON(streams.Out, map[string]any{
							"movie_id":   mid,
							"best":       b,
							"magnet_uri": javdb.MagnetURI(b),
						})
					}
					if b == nil {
						fmt.Fprintln(streams.Err, "(无磁力链)")
						return nil
					}
					writeMagnets(streams.Out, streams.Err, []map[string]any{b})
					return nil
				}
				if asJSON {
					return writeJSON(streams.Out, map[string]any{"movie_id": mid, "magnets": items})
				}
				writeMagnets(streams.Out, streams.Err, items)
				return nil
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "magnets NUMBER",
		Short: "List magnet links for a movie",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runner.Context = cmd.Context()
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&cnsub, "cnsub", false, "Only magnets with Chinese subtitles")
	cmd.Flags().BoolVar(&hd, "hd", false, "Only HD magnets")
	cmd.Flags().StringVar(&minSize, "min-size", "", "Min size e.g. 2000, 4GB, 500MB")
	cmd.Flags().BoolVar(&best, "best", false, "Pick single best magnet (cnsub > hd > size)")
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}

// renderText 将磁力信封投影为可继续传给下游的 magnet URI；不把人类表格列
// 混入 stdout，也不吞写入错误。
func renderText(w io.Writer, envelope pipeline.Envelope, best bool) error {
	if best {
		uri, _ := envelope.Data["magnet_uri"].(string)
		if uri == "" {
			return nil
		}
		_, err := fmt.Fprintln(w, uri)
		return err
	}
	magnets, _ := envelope.Data["magnets"].([]map[string]any)
	for _, magnet := range magnets {
		uri := javdb.MagnetURI(magnet)
		if uri == "" {
			continue
		}
		if _, err := fmt.Fprintln(w, uri); err != nil {
			return err
		}
	}
	return nil
}

// ParseSizeMiB parses the CLI magnet size filter without changing its unit rules.
func ParseSizeMiB(text string) (int, error) {
	s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(text), " ", ""))
	if s == "" {
		return 0, nil
	}
	mult := 1.0
	switch {
	case strings.HasSuffix(s, "GB"):
		s = strings.TrimSuffix(s, "GB")
		mult = 1024
	case strings.HasSuffix(s, "G"):
		s = strings.TrimSuffix(s, "G")
		mult = 1024
	case strings.HasSuffix(s, "MB"):
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "M"):
		s = strings.TrimSuffix(s, "M")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid --min-size: %s", text)
	}
	return int(f * mult), nil
}

// magnetCount 是 magnets_count 的整数解析（缺失 → 0）。
func magnetCount(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		if n, ok := scalar.Int64(v); ok {
			return int(n)
		}
		return 0
	}
}

// knownMagnetCount 从输入信封中提取已知的 magnets_count（上游 search 信封的
// data.movie 携带该字段时可用）。返回 (count, true) 表示已知，(0, false) 表示
// 未知，需要请求 MovieDetail。
func knownMagnetCount(input pipeline.Envelope) (int, bool) {
	if input.Data == nil {
		return 0, false
	}
	movie, ok := input.Data["movie"].(map[string]any)
	if !ok {
		return 0, false
	}
	raw, exists := movie["magnets_count"]
	if !exists {
		return 0, false
	}
	return magnetCount(raw), true
}
