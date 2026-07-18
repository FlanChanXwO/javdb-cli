package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
)

type entityFlags struct {
	zone, sort, order        string
	page, limit              int
	tagRefs, mainFlags       []string
	allPages, hasMagnets     bool
	asJSON                   bool
}

func addEntityFlags(cmd *cobra.Command, f *entityFlags) {
	cmd.Flags().StringVar(&f.zone, "zone", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringArrayVar(&f.tagRefs, "tag", nil, "Content tag id/EN/中文 (repeatable)")
	cmd.Flags().StringArrayVar(&f.mainFlags, "main", nil, "Main flag p|m|c|s|i|v (repeatable)")
	cmd.Flags().StringVar(&f.sort, "sort", "release", "hit|release|score|update|want_watch_count|watched_count")
	cmd.Flags().StringVar(&f.order, "order", "desc", "asc|desc")
	cmd.Flags().IntVar(&f.page, "page", 1, "Page")
	cmd.Flags().IntVar(&f.limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&f.allPages, "all", false, "Fetch all pages (capped)")
	cmd.Flags().BoolVar(&f.hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&f.asJSON, "json", false, "JSON with entity meta + movies")
}

func runEntity(rf *rootFlags, aio *appIO, kind, ref string, f entityFlags) error {
	rt, err := loadRuntime(rf)
	if err != nil {
		return err
	}
	c, err := newClient(rt, "")
	if err != nil {
		return err
	}
	ctx := context.Background()
	eid, err := c.ResolveEntity(ctx, kind, ref, f.zone)
	if err != nil {
		return fmt.Errorf("%s failed: %w", kind, err)
	}
	var tagIDs []string
	if len(f.tagRefs) > 0 {
		tagIDs, err = c.ResolveTags(ctx, f.tagRefs, f.zone)
		if err != nil {
			return fmt.Errorf("%s failed: %w", kind, err)
		}
	}
	opt := appapi.EntityMoviesOptions{
		Zone: f.zone, Page: f.page, Limit: f.limit,
		Sort: f.sort, Order: f.order, Main: f.mainFlags, Tags: tagIDs,
	}
	var movies []map[string]any
	if f.allPages {
		movies, err = c.AllEntityMovies(ctx, kind, eid, opt, 50)
		if err != nil {
			return fmt.Errorf("%s failed: %w", kind, err)
		}
	} else {
		res, err := c.EntityMovies(ctx, kind, eid, opt)
		if err != nil {
			return fmt.Errorf("%s failed: %w", kind, err)
		}
		movies = res.Movies()
	}
	if f.hasMagnets {
		movies = FilterHasMagnets(movies)
	}
	var meta map[string]any
	if m, err := c.EntityDetail(ctx, kind, eid); err == nil {
		meta = m
	} else {
		meta = map[string]any{"id": eid}
	}
	if f.asJSON {
		return EmitJSON(aio.out, map[string]any{
			"entity": meta, "entity_id": eid, "movies": movies,
		})
	}
	PrintMovies(aio.out, aio.err, movies)
	return nil
}

func newEntityCmd(rf *rootFlags, aio *appIO, kind, use, short string) *cobra.Command {
	f := entityFlags{}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEntity(rf, aio, kind, args[0], f)
		},
	}
	addEntityFlags(cmd, &f)
	return cmd
}

func registerEntityCmds(root *cobra.Command, rf *rootFlags, aio *appIO) {
	root.AddCommand(newEntityCmd(rf, aio, "actor", "actor REF", "List movies for an actor (id or name)"))
	root.AddCommand(newEntityCmd(rf, aio, "series", "series REF", "List movies for a series (id or name)"))
	root.AddCommand(newEntityCmd(rf, aio, "maker", "maker REF", "List movies for a maker/studio (id or name)"))
	root.AddCommand(newEntityCmd(rf, aio, "director", "director REF", "List movies for a director (id or name)"))
	root.AddCommand(newEntityCmd(rf, aio, "code", "code REF", "List movies for a code/prefix e.g. SSIS"))
	root.AddCommand(newEntityCmd(rf, aio, "list", "list REF", "List movies inside a 合集 (user playlist)"))
}
