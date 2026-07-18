package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func newMagnetsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var (
		cnsub, hd, best, isID, asJSON bool
		minSize                       string
	)
	cmd := &cobra.Command{
		Use:   "magnets NUMBER",
		Short: "List magnet links for a movie (needs login)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			tok, err := defaultToken()
			if err != nil {
				return err
			}
			c, err := newClient(rt, tok)
			if err != nil {
				return err
			}
			ctx := context.Background()
			mid := args[0]
			if !isID {
				mid, err = c.ResolveMovieID(ctx, args[0])
				if err != nil {
					return err
				}
			}
			// optional early exit if magnets_count known zero
			detail, err := c.MovieDetail(ctx, mid)
			if err != nil {
				return fmt.Errorf("magnets failed: %w", err)
			}
			var items []map[string]any
			if anyToInt(detail["magnets_count"]) == 0 {
				items = nil
			} else {
				items, err = c.MovieMagnets(ctx, mid)
				if err != nil {
					return fmt.Errorf("magnets failed: %w", err)
				}
			}
			minMiB := 0
			if minSize != "" {
				minMiB, err = parseSizeMiB(minSize)
				if err != nil {
					return err
				}
			}
			items = javdb.FilterMagnets(items, cnsub, hd, minMiB)
			if best {
				b := javdb.PickBestMagnet(items)
				if asJSON {
					return EmitJSON(aio.out, map[string]any{
						"movie_id":   mid,
						"best":       b,
						"magnet_uri": javdb.MagnetURI(b),
					})
				}
				if b == nil {
					fmt.Fprintln(aio.err, "(无磁力链)")
					return nil
				}
				PrintMovieMagnets(aio.out, aio.err, []map[string]any{b})
				return nil
			}
			if asJSON {
				return EmitJSON(aio.out, map[string]any{"movie_id": mid, "magnets": items})
			}
			PrintMovieMagnets(aio.out, aio.err, items)
			return nil
		},
	}
	cmd.Flags().BoolVar(&cnsub, "cnsub", false, "Only magnets with Chinese subtitles")
	cmd.Flags().BoolVar(&hd, "hd", false, "Only HD magnets")
	cmd.Flags().StringVar(&minSize, "min-size", "", "Min size e.g. 2000, 4GB, 500MB")
	cmd.Flags().BoolVar(&best, "best", false, "Pick single best magnet (cnsub > hd > size)")
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}

func parseSizeMiB(text string) (int, error) {
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
