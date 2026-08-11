// Package history 负责显式同步历史 GitHub Release 正文。
package history

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/document"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/github"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

// Config 是历史 Release 同步配置。
type Config struct {
	Repository string
	Version    string
	Directory  string
	Client     github.Client
	Apply      bool
}

// Run 执行 sync-history 子命令。
func Run(arguments []string) error {
	flags := flag.NewFlagSet("sync-history", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	repository := flags.String("repo", "", "GitHub repository in owner/name form")
	version := flags.String("version", "", "semantic version without v")
	directory := flags.String("dir", "", "version directory containing en.md and zh-CN.md")
	apiBase := flags.String("api-base", "https://api.github.com", "GitHub REST API base URL")
	tokenEnv := flags.String("token-env", "GH_TOKEN", "environment variable containing a GitHub token")
	apply := flags.Bool("apply", false, "create or update the historical GitHub Release body")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || !github.ValidRepository(*repository) || !document.IsSemanticVersion(*version) || *directory == "" {
		return errors.New("sync-history requires --repo, --version, and --dir")
	}
	return SyncHistoricalRelease(context.Background(), Config{
		Repository: *repository,
		Version:    *version,
		Directory:  *directory,
		Apply:      *apply,
		Client:     github.New(*apiBase, os.Getenv(*tokenEnv), http.DefaultClient),
	})
}

// SyncHistoricalRelease 只读取或更新 Release 正文。它不接触 tag、draft 状态和
// assets，因此补写历史文本不会补造或替换任何已发布的二进制资产。
func SyncHistoricalRelease(ctx context.Context, config Config) error {
	body, err := document.RenderGitHubReleaseBody(config.Directory, config.Version)
	if err != nil {
		return err
	}
	path := "/repos/" + config.Repository + "/releases/tags/v" + config.Version
	var release model.GitHubRelease
	err = config.Client.GetJSON(ctx, path, &release)
	if err != nil {
		var apiErr *github.HTTPError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return err
		}
		if !config.Apply {
			fmt.Fprintf(os.Stdout, "would create historical Release v%s without assets\n", config.Version)
			return nil
		}
		created := model.GitHubRelease{}
		if err := config.Client.RequestJSON(ctx, http.MethodPost, "/repos/"+config.Repository+"/releases", model.GitHubReleaseWrite{
			TagName: "v" + config.Version,
			Name:    "v" + config.Version,
			Body:    body,
			Draft:   false,
		}, &created); err != nil {
			return fmt.Errorf("create historical Release v%s: %w", config.Version, err)
		}
		release = created
	} else {
		if !config.Apply {
			fmt.Fprintf(os.Stdout, "would update historical Release v%s body; assets remain unchanged\n", config.Version)
			return nil
		}
		if err := config.Client.RequestJSON(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/releases/%d", config.Repository, release.ID), model.GitHubReleaseWrite{Body: body}, &release); err != nil {
			return fmt.Errorf("update historical Release v%s: %w", config.Version, err)
		}
	}
	var verified model.GitHubRelease
	if err := config.Client.GetJSON(ctx, path, &verified); err != nil {
		return fmt.Errorf("verify historical Release v%s: %w", config.Version, err)
	}
	if verified.TagName != "v"+config.Version {
		return fmt.Errorf("verified historical Release tag %q, want v%s", verified.TagName, config.Version)
	}
	if verified.Body != body {
		return fmt.Errorf("verified historical Release v%s body differs from local rendering", config.Version)
	}
	return nil
}
