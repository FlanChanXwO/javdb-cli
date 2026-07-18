package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

// PrintRankedMovies prefixes each row with #ranking.
func PrintRankedMovies(w io.Writer, errW io.Writer, movies []map[string]any) {
	if len(movies) == 0 {
		fmt.Fprintln(errW, "(空列表)")
		return
	}
	for _, m := range movies {
		rank := anyString(m["ranking"])
		line := fmt.Sprintf("#%s\t%s\t%s\t%s", rank, anyString(m["number"]), anyString(m["id"]), anyString(m["title"]))
		if d := anyString(m["release_date"]); d != "" {
			line += "\t" + d
		}
		fmt.Fprintln(w, line)
	}
}

// PrintNamedNoCount prints id\tname only.
func PrintNamedNoCount(w io.Writer, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, "(空列表)")
		return
	}
	for _, it := range items {
		name := anyString(it["name_zht"])
		if name == "" {
			name = anyString(it["name"])
		}
		fmt.Fprintf(w, "%s\t%s\n", anyString(it["id"]), name)
	}
}

func newRankingsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rankings",
		Short: "Movie/actor rankings (no login needed)",
	}
	cmd.AddCommand(newRankingsMoviesCmd(rf, aio))
	cmd.AddCommand(newRankingsActorsCmd(rf, aio))
	cmd.AddCommand(newRankingsPlaybackCmd(rf, aio))
	return cmd
}

func newRankingsMoviesCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var type_, period string
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "movies",
		Short: "Movie rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsMovies(context.Background(), type_, period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = FilterHasMagnets(movies)
			}
			PrintMovies(aio.out, aio.err, movies)
			return nil
		},
	}
	cmd.Flags().StringVar(&type_, "type", "censored", "censored|uncensored|western")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func newRankingsActorsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var period string
	cmd := &cobra.Command{
		Use:   "actors",
		Short: "Actor rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsActors(context.Background(), javdb.ActorPeriod(period))
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			PrintNamedNoCount(aio.out, aio.err, res.Named("actors"))
			return nil
		},
	}
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	return cmd
}

func newRankingsPlaybackCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var filterBy, period string
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "playback",
		Short: "Playback rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsPlayback(context.Background(), filterBy, period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = FilterHasMagnets(movies)
			}
			PrintMovies(aio.out, aio.err, movies)
			return nil
		},
	}
	cmd.Flags().StringVar(&filterBy, "filter-by", "censored", "censored|uncensored|western")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func newTop250Cmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var zone, year string
	var startRank, page, limit int
	var ignoreWatched, hasMagnets bool
	cmd := &cobra.Command{
		Use:   "top250",
		Short: "TOP250 list (needs login)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				res, err := c.Top250(context.Background(), zone, year, startRank, page, limit, ignoreWatched)
				if err != nil {
					return fmt.Errorf("top250 failed: %w", err)
				}
				if gen := resRawString(res, "generated_at"); gen != "" {
					fmt.Fprintf(aio.err, "# generated_at=%s\n", gen)
				}
				movies := res.Movies()
				if hasMagnets {
					movies = FilterHasMagnets(movies)
				}
				PrintRankedMovies(aio.out, aio.err, movies)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", "censored|uncensored|western|fc2 (omit for all-site)")
	cmd.Flags().StringVar(&year, "year", "", "Filter by year e.g. 2023")
	cmd.Flags().IntVar(&startRank, "from", 1, "Start from this rank")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&ignoreWatched, "ignore-watched", false, "Skip already watched titles")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func resRawString(res javdb.SearchResult, key string) string {
	raw, ok := res[key]
	if !ok || len(raw) == 0 {
		return ""
	}
	s := string(raw)
	if len(s) >= 2 && s[0] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
