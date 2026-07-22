package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

// PrintLists writes id\tname\tmovies\tprivacy\tviews.
func PrintLists(w io.Writer, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, "(空列表)")
		return
	}
	for _, it := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			anyString(it["id"]),
			anyString(it["name"]),
			anyString(it["movies_count"]),
			anyString(it["privacy"]),
			anyString(it["views_count"]),
		)
	}
}

func newListsCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var page, limit int
	var sortBy string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "lists",
		Short: "My 合集; subcommands: show/search/related",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withAuthedClient(rf, aio, func(c *javdb.Client) error {
				res, err := c.MyLists(context.Background(), page, limit, sortBy)
				if err != nil {
					return fmt.Errorf("lists failed: %w", err)
				}
				items := res.Named("lists")
				if asJSON {
					return EmitJSON(aio.out, map[string]any{
						"lists":        items,
						"current_page": resRawString(res, "current_page"),
					})
				}
				PrintLists(aio.out, aio.err, items)
				return nil
			})
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().StringVar(&sortBy, "sort-by", "created", "created|name|movies_count|views_count|updated|default")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")

	cmd.AddCommand(newListsShowCmd(rf, aio))
	cmd.AddCommand(newListsSearchCmd(rf, aio))
	cmd.AddCommand(newListsRelatedCmd(rf, aio))
	return cmd
}

func newListsShowCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show REF",
		Short: "Show 合集 meta (movies: use list <id>)",
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
			ctx := context.Background()
			eid, err := c.ResolveEntity(ctx, "list", args[0], "censored")
			if err != nil {
				return fmt.Errorf("lists show failed: %w", err)
			}
			data, err := c.ListInfo(ctx, eid)
			if err != nil {
				return fmt.Errorf("lists show failed: %w", err)
			}
			if asJSON {
				return EmitJSON(aio.out, data)
			}
			meta, _ := data["list"].(map[string]any)
			if meta == nil {
				if m, ok := data["list"]; ok {
					b, _ := json.Marshal(m)
					_ = json.Unmarshal(b, &meta)
				}
			}
			if meta == nil {
				meta = data
			}
			fmt.Fprintf(aio.out, "id\t%s\n", coalesce(anyString(meta["id"]), eid))
			fmt.Fprintf(aio.out, "name\t%s\n", anyString(meta["name"]))
			if d := anyString(meta["description"]); d != "" {
				fmt.Fprintf(aio.out, "desc\t%s\n", d)
			}
			fmt.Fprintf(aio.out, "movies\t%s\n", anyString(meta["movies_count"]))
			fmt.Fprintf(aio.out, "views\t%s\n", anyString(meta["views_count"]))
			fmt.Fprintf(aio.out, "collects\t%s\n", anyString(meta["collections_count"]))
			if s := anyString(meta["share_info"]); s != "" {
				fmt.Fprintf(aio.out, "share\t%s\n", s)
			}
			fmt.Fprintf(aio.out, "is_creator\t%v\n", data["is_creator"])
			fmt.Fprintf(aio.out, "has_collected\t%v\n", data["has_collected"])
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func newListsSearchCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var page, limit int
	var zone string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search KEYWORD",
		Short: "Search public 合集",
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
			res, err := c.Search(context.Background(), args[0], javdb.SearchOptions{
				Page: page, Limit: limit, Zone: zone, Type: "list",
			})
			if err != nil {
				return fmt.Errorf("lists search failed: %w", err)
			}
			items := res.Named("lists")
			if asJSON {
				return EmitJSON(aio.out, map[string]any{"lists": items})
			}
			PrintLists(aio.out, aio.err, items)
			return nil
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size")
	cmd.Flags().StringVar(&zone, "zone", "all", "censored|uncensored|western|fc2|all")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func newListsRelatedCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var isID bool
	var page, limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "related NUMBER",
		Short: "Public 合集 related to a movie",
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
			ctx := context.Background()
			mid := args[0]
			if !isID {
				mid, err = c.ResolveMovieID(ctx, args[0])
				if err != nil {
					return err
				}
			}
			res, err := c.RelatedLists(ctx, mid, page, limit)
			if err != nil {
				return fmt.Errorf("lists related failed: %w", err)
			}
			items := res.Named("lists")
			if asJSON {
				return EmitJSON(aio.out, map[string]any{"lists": items})
			}
			PrintLists(aio.out, aio.err, items)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat NUMBER as internal movie id")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&asJSON, "json", false, "JSON output")
	return cmd
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
