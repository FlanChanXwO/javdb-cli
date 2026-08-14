// Package magnets 提供影片磁力列表与最佳磁力选择命令。
package magnets

import (
	"context"
	"fmt"
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
		cnsub, hd, best, isID   bool
		asJSON, asJSONL, asText bool
		minSize                 string
	)
	fetch := func(c *javdb.Client, ctx context.Context, ref string, useID bool) (string, []map[string]any, error) {
		mid := ref
		var err error
		if !useID {
			mid, err = c.ResolveMovieID(ctx, ref)
			if err != nil {
				return "", nil, err
			}
		}
		detail, err := c.MovieDetail(ctx, mid)
		if err != nil {
			return "", nil, fmt.Errorf("magnets failed: %w", err)
		}
		var items []map[string]any
		if magnetCount(detail["magnets_count"]) == 0 {
			items = nil
		} else {
			items, err = c.MovieMagnets(ctx, mid)
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
	runner := &pipeline.BatchRunner{
		Name:       "magnets",
		LegacyJSON: true,
		Kinds:      []pipeline.Kind{pipeline.KindMovie},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			ref := pipeline.ConsumerRef(input)
			mid, items, err := fetch(c, ctx, ref, isID && input.ID == "")
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
		},
		Legacy: func(args []string) error {
			return client.WithOptionalAuth(options, streams.Err, func(c *javdb.Client) error {
				ctx := context.Background()
				mid, items, err := fetch(c, ctx, args[0], isID)
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
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&cnsub, "cnsub", false, "Only magnets with Chinese subtitles")
	cmd.Flags().BoolVar(&hd, "hd", false, "Only HD magnets")
	cmd.Flags().StringVar(&minSize, "min-size", "", "Min size e.g. 2000, 4GB, 500MB")
	cmd.Flags().BoolVar(&best, "best", false, "Pick single best magnet (cnsub > hd > size)")
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default)")
	return cmd
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
