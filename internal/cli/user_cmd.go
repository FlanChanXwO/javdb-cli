package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func newWatchedCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "watched",
		Short: "List watched (看過) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				movies, err := c.WatchedMovies(context.Background())
				if err != nil {
					return err
				}
				if hasMagnets {
					movies = FilterHasMagnets(movies)
				}
				PrintMovies(aio.out, aio.err, movies)
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func newWantCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "want",
		Short: "List want-to-watch (想看) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				movies, err := c.WantMovies(context.Background())
				if err != nil {
					return err
				}
				if hasMagnets {
					movies = FilterHasMagnets(movies)
				}
				PrintMovies(aio.out, aio.err, movies)
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func newRecentCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recently viewed (最近浏览) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				movies, err := c.RecentViewed(context.Background())
				if err != nil {
					return err
				}
				if hasMagnets {
					movies = FilterHasMagnets(movies)
				}
				PrintMovies(aio.out, aio.err, movies)
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func newCollectionsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	return &cobra.Command{
		Use:   "collections KIND",
		Short: "List a collection: actors|series|codes|makers|directors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := args[0]
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				items, err := c.Collected(context.Background(), kind)
				if err != nil {
					return err
				}
				PrintNamed(aio.out, aio.err, items)
				return nil
			})
		},
	}
}

func newMarkCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var watched, want, isID bool
	var score int
	var content string
	cmd := &cobra.Command{
		Use:   "mark NUMBER",
		Short: "Mark a movie as 看過 (--watched) or 想看 (--want)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if watched == want {
				return fmt.Errorf("specify exactly one of --watched or --want")
			}
			status := "want_watch"
			label := "想看"
			if watched {
				status = "watched"
				label = "看過"
			}
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				ctx := context.Background()
				mid := args[0]
				var err error
				if !isID {
					mid, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				rev, err := c.Mark(ctx, mid, status, score, content)
				if err != nil {
					return fmt.Errorf("mark failed: %w", err)
				}
				fmt.Fprintf(aio.out, "已标记 %s (%s) → %s\treview_id=%s\n",
					args[0], mid, label, anyString(rev["id"]))
				return nil
			})
		},
	}
	cmd.Flags().BoolVar(&watched, "watched", false, "Mark as 看過")
	cmd.Flags().BoolVar(&want, "want", false, "Mark as 想看")
	cmd.Flags().IntVar(&score, "score", 0, "Optional score")
	cmd.Flags().StringVar(&content, "content", "", "Optional review text")
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	return cmd
}

func newUnmarkCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var isID bool
	cmd := &cobra.Command{
		Use:   "unmark NUMBER",
		Short: "Remove watched/want mark for a movie",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				ctx := context.Background()
				mid := args[0]
				var err error
				if !isID {
					mid, err = c.ResolveMovieID(ctx, args[0])
					if err != nil {
						return err
					}
				}
				ok, err := c.Unmark(ctx, mid)
				if err != nil {
					return fmt.Errorf("unmark failed: %w", err)
				}
				if ok {
					fmt.Fprintf(aio.out, "已取消标记 %s (%s)\n", args[0], mid)
				} else {
					fmt.Fprintf(aio.out, "无标记可取消 %s (%s)\n", args[0], mid)
				}
				return nil
			})
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	return cmd
}

func registerUserCmds(root *cobra.Command, rf *rootFlags, aio *appIO) {
	root.AddCommand(newWatchedCmd(rf, aio))
	root.AddCommand(newWantCmd(rf, aio))
	root.AddCommand(newRecentCmd(rf, aio))
	root.AddCommand(newCollectionsCmd(rf, aio))
	root.AddCommand(newMarkCmd(rf, aio))
	root.AddCommand(newUnmarkCmd(rf, aio))
}
