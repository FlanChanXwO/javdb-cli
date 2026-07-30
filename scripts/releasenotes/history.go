// 历史 Release 同步是 releasenotes 中唯一写入 GitHub 数据的命令路径。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotesrender"
)

func runSyncHistory(arguments []string) error {
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
	if flags.NArg() != 0 || !validRepository(*repository) || !semanticVersionPattern.MatchString(*version) || *directory == "" {
		return errors.New("sync-history requires --repo, --version, and --dir")
	}
	return syncHistoricalRelease(context.Background(), syncHistoryConfig{
		Repository: *repository,
		Version:    *version,
		Directory:  *directory,
		Apply:      *apply,
		Client: githubClient{
			baseURL: *apiBase,
			token:   os.Getenv(*tokenEnv),
			client:  http.DefaultClient,
		},
	})
}

// syncHistoricalRelease 只读取或更新 Release 正文。它不接触 tag、draft 状态和
// assets，因此补写历史文本不会补造或替换任何已发布的二进制资产。
func syncHistoricalRelease(ctx context.Context, config syncHistoryConfig) error {
	body, err := renderGitHubReleaseBody(config.Directory, config.Version)
	if err != nil {
		return err
	}
	path := "/repos/" + config.Repository + "/releases/tags/v" + config.Version
	var release githubRelease
	err = config.Client.getJSON(ctx, path, &release)
	if err != nil {
		var apiErr *githubHTTPError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return err
		}
		if !config.Apply {
			fmt.Fprintf(os.Stdout, "would create historical Release v%s without assets\n", config.Version)
			return nil
		}
		created := githubRelease{}
		if err := config.Client.requestJSON(ctx, http.MethodPost, "/repos/"+config.Repository+"/releases", githubReleaseWrite{
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
		if err := config.Client.requestJSON(ctx, http.MethodPatch, fmt.Sprintf("/repos/%s/releases/%d", config.Repository, release.ID), githubReleaseWrite{Body: body}, &release); err != nil {
			return fmt.Errorf("update historical Release v%s: %w", config.Version, err)
		}
	}
	var verified githubRelease
	if err := config.Client.getJSON(ctx, path, &verified); err != nil {
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

func renderGitHubReleaseBody(directory, version string) (string, error) {
	english, err := releasenotesrender.NotesFromChangelog(filepath.Join(directory, "en.md"), version)
	if err != nil {
		return "", fmt.Errorf("English changelog: %w", err)
	}
	chinese, err := releasenotesrender.NotesFromChangelog(filepath.Join(directory, "zh-CN.md"), version)
	if err != nil {
		return "", fmt.Errorf("Simplified Chinese changelog: %w", err)
	}
	return string(releasenotesrender.BilingualBody(english, chinese)), nil
}
