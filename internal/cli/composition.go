package cli

import (
	"github.com/spf13/cobra"

	actorcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/actor"
	assetscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/assets"
	authcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/auth"
	browsecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/browse"
	cachecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/cache"
	codecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/code"
	collectionscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/collections"
	commentscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/comments"
	configcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/config"
	detailcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/detail"
	directorcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/director"
	listcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/list"
	listscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/lists"
	magnetscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/magnets"
	makercmd "github.com/FlanChanXwO/javdb-cli/internal/cli/maker"
	markcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/mark"
	rankingscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/rankings"
	recentcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/recent"
	searchcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/search"
	seriescmd "github.com/FlanChanXwO/javdb-cli/internal/cli/series"
	tagscmd "github.com/FlanChanXwO/javdb-cli/internal/cli/tags"
	top250cmd "github.com/FlanChanXwO/javdb-cli/internal/cli/top250"
	unmarkcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/unmark"
	updatecmd "github.com/FlanChanXwO/javdb-cli/internal/cli/update"
	wantcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/want"
	watchedcmd "github.com/FlanChanXwO/javdb-cli/internal/cli/watched"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

func addCommands(command *cobra.Command, options *session.Options, streams *environment.Streams) {
	// 保持原 root.go 的 AddCommand 顺序，避免 help 与 completion 输出漂移。
	command.AddCommand(authcmd.New(options, streams))
	command.AddCommand(configcmd.New(streams))
	command.AddCommand(cachecmd.New(streams))
	command.AddCommand(searchcmd.New(options, streams))
	command.AddCommand(detailcmd.New(options, streams))
	command.AddCommand(commentscmd.New(options, streams))
	command.AddCommand(magnetscmd.New(options, streams))
	command.AddCommand(tagscmd.New(options, streams))
	command.AddCommand(browsecmd.New(options, streams))
	command.AddCommand(actorcmd.New(options, streams))
	command.AddCommand(assetscmd.New(options, streams))
	command.AddCommand(seriescmd.New(options, streams))
	command.AddCommand(makercmd.New(options, streams))
	command.AddCommand(directorcmd.New(options, streams))
	command.AddCommand(codecmd.New(options, streams))
	command.AddCommand(listcmd.New(options, streams))
	command.AddCommand(watchedcmd.New(options, streams))
	command.AddCommand(wantcmd.New(options, streams))
	command.AddCommand(recentcmd.New(options, streams))
	command.AddCommand(collectionscmd.New(options, streams))
	command.AddCommand(markcmd.New(options, streams))
	command.AddCommand(unmarkcmd.New(options, streams))
	command.AddCommand(rankingscmd.New(options, streams))
	command.AddCommand(top250cmd.New(options, streams))
	command.AddCommand(listscmd.New(options, streams))
	command.AddCommand(updatecmd.New(options, streams))
}
