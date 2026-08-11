package auth

import (
	"fmt"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"strconv"

	"github.com/spf13/cobra"
)

// NewUse builds the auth use command.
func NewUse(streams *invocation.Streams) *cobra.Command {
	return &cobra.Command{
		Use:   "use <user_id>",
		Short: "Set the default account",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			userID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("user_id must be integer: %w", err)
			}
			// 参数校验通过后再按"第一个真实命令创建配置"契约触发首次创建。
			if err := paths.EnsureDefaultConfigFile(); err != nil {
				return err
			}
			fileStore, store, err := authstore.Open()
			if err != nil {
				return err
			}
			if err := store.Use(userID); err != nil {
				return err
			}
			if err := fileStore.Commit(store); err != nil {
				return err
			}
			account, _ := store.Get(userID)
			_, err = fmt.Fprintf(streams.Out, "default account → %s (id=%d)\n", account.Username, userID)
			return err
		},
	}
}
