package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func newSearchCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var (
		page, limit   int
		zone, sort    string
		filterBy, typ string
		hasMagnets    bool
		asJSON        bool
	)
	cmd := &cobra.Command{
		Use:   "search KEYWORD",
		Short: "Search movies (or other dimensions with --type)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			c, err := newClient(rt, "")
			if err != nil {
				return err
			}
			opt := appapi.SearchOptions{
				Page:     page,
				Limit:    limit,
				Zone:     zone,
				Sort:     sort,
				FilterBy: filterBy,
				Type:     typ,
			}
			res, err := c.Search(context.Background(), args[0], opt)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}
			return renderSearch(aio, res, typ, hasMagnets, asJSON)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (0 = server default)")
	cmd.Flags().StringVar(&zone, "zone", "censored", "censored|uncensored|western|fc2|all")
	cmd.Flags().StringVar(&sort, "sort", "", "relevance|release|score|update|hit")
	cmd.Flags().StringVar(&filterBy, "filter-by", "", "can_play|magnets|subtitle|single")
	cmd.Flags().StringVar(&typ, "type", "", "movie|code|series|actor|maker|director|list")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop movie rows with magnets_count == 0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}

// renderSearch mirrors Python search output selection:
// movies path when type is empty/movie OR response carries movies key for default type.
func renderSearch(aio *appIO, res javdb.SearchResult, typ string, hasMagnets, asJSON bool) error {
	if typ == "" || typ == "movie" {
		movies := res.Movies()
		if hasMagnets {
			movies = FilterHasMagnets(movies)
		}
		if asJSON {
			return EmitJSON(aio.out, map[string]any{"movies": movies})
		}
		PrintMovies(aio.out, aio.err, movies)
		return nil
	}
	key := SearchTypeKey(typ)
	items := res.Named(key)
	if asJSON {
		return EmitJSON(aio.out, map[string]any{key: items})
	}
	PrintNamed(aio.out, aio.err, items)
	return nil
}
