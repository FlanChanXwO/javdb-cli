package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func newDetailCmd(rf *rootFlags, aio *appIO) *cobra.Command {
	var isID, withMagnets, asJSON bool
	cmd := &cobra.Command{
		Use:   "detail NUMBER",
		Short: "Show movie detail (graph ids for agent navigation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := loadRuntime(rf)
			if err != nil {
				return err
			}
			token := ""
			if withMagnets {
				// magnets require auth
				tok, err := defaultToken()
				if err != nil {
					return err
				}
				token = tok
			}
			c, err := newClient(rt, token)
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
			movie, err := c.MovieDetail(ctx, mid)
			if err != nil {
				return fmt.Errorf("detail failed: %w", err)
			}
			var mags []map[string]any
			if withMagnets {
				mags, err = c.MovieMagnets(ctx, mid)
				if err != nil {
					return fmt.Errorf("magnets failed: %w", err)
				}
			}
			if asJSON {
				payload := map[string]any{}
				for k, v := range movie {
					payload[k] = v
				}
				if withMagnets {
					payload["magnets"] = mags
				}
				return EmitJSON(aio.out, payload)
			}
			PrintDetail(aio.out, movie)
			if withMagnets {
				PrintMovieMagnets(aio.out, aio.err, mags)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, "Treat argument as internal movie id")
	cmd.Flags().BoolVar(&withMagnets, "magnets", false, "Also list magnet links (needs login)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	return cmd
}

// defaultToken loads the default account JWT from auth store.
func defaultToken() (string, error) {
	_, store, err := openAuth()
	if err != nil {
		return "", err
	}
	acc, err := store.Default()
	if err != nil {
		return "", fmt.Errorf("no default account; run: javdb auth login")
	}
	if acc.Token == "" {
		return "", fmt.Errorf("default account has no token; run: javdb auth login")
	}
	return acc.Token, nil
}

// ensure javdb import used if only types — already used via newClient
var _ = javdb.HostMirror
