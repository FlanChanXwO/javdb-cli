// Package audit 负责读取 GitHub 元数据并生成发布区间的来源审计报告。
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/github"
	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

// Config 是 audit 收集流程的配置。
type Config struct {
	Repository string
	From       string
	To         string
	Client     github.Client
}

var squashPullRequestPattern = regexp.MustCompile(`\(#([1-9][0-9]*)\)\s*$`)

// Run 执行 audit 子命令。
func Run(arguments []string) error {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	repository := flags.String("repo", "", "GitHub repository in owner/name form")
	from := flags.String("from", "", "exclusive base tag; empty for the initial release")
	to := flags.String("to", "", "inclusive Git ref or tag")
	apiBase := flags.String("api-base", "https://api.github.com", "GitHub REST API base URL")
	tokenEnv := flags.String("token-env", "GH_TOKEN", "environment variable containing an optional GitHub token")
	output := flags.String("output", "", "optional JSON report path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("audit accepts no positional arguments: %q", flags.Arg(0))
	}
	if !github.ValidRepository(*repository) || *to == "" {
		return errors.New("audit requires --repo owner/name and --to")
	}
	report, err := Collect(context.Background(), Config{
		Repository: *repository,
		From:       *from,
		To:         *to,
		Client:     github.New(*apiBase, os.Getenv(*tokenEnv), http.DefaultClient),
	})
	if err != nil {
		return err
	}
	return WriteJSON(*output, report)
}

// Collect 收集发布区间内的 commit 和 PR 来源。
func Collect(ctx context.Context, config Config) (model.AuditReport, error) {
	commits, err := gitRevisionList(ctx, config.From, config.To)
	if err != nil {
		return model.AuditReport{}, err
	}
	report := model.AuditReport{
		Repository: config.Repository,
		From:       config.From,
		To:         config.To,
		Sources:    make([]model.AuditSource, 0),
	}
	seenPulls := make(map[int]struct{})
	for _, commit := range commits {
		title, author, detailErr := gitCommitDetail(ctx, commit)
		if detailErr != nil {
			return model.AuditReport{}, detailErr
		}
		pulls, err := pullRequestsForCommit(ctx, config.Client, config.Repository, commit, title)
		if err != nil {
			return model.AuditReport{}, err
		}
		if len(pulls) == 0 {
			report.Sources = append(report.Sources, model.AuditSource{
				Kind:   "commit",
				URL:    "https://github.com/" + config.Repository + "/commit/" + commit,
				Commit: commit,
				Title:  title,
				Author: author,
				Issue:  "direct commit requires an explicit historical attribution",
			})
			continue
		}
		for _, summary := range pulls {
			if _, exists := seenPulls[summary.Number]; exists {
				continue
			}
			seenPulls[summary.Number] = struct{}{}
			pull, err := config.Client.PullRequest(ctx, config.Repository, summary.Number)
			if err != nil {
				return model.AuditReport{}, fmt.Errorf("lookup PR #%d: %w", summary.Number, err)
			}
			report.Sources = append(report.Sources, model.AuditSource{
				Kind:       "pull_request",
				URL:        pull.HTMLURL,
				PullNumber: pull.Number,
				Title:      pull.Title,
				Author:     pull.User.Login,
			})
		}
	}
	sort.Slice(report.Sources, func(left, right int) bool { return report.Sources[left].URL < report.Sources[right].URL })
	return report, nil
}

// WriteJSON 将值格式化为稳定的审计 JSON；空路径写 stdout，指定路径使用 0600。
func WriteJSON(path string, value any) error {
	var destination io.Writer = os.Stdout
	var file *os.File
	if path != "" {
		opened, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		file = opened
		destination = file
	}
	encoder := json.NewEncoder(destination)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		if file != nil {
			_ = file.Close()
		}
		return err
	}
	if file != nil {
		return file.Close()
	}
	return nil
}

func gitRevisionList(ctx context.Context, from, to string) ([]string, error) {
	if to == "" {
		return nil, errors.New("git revision target is required")
	}
	revision := to
	if from != "" {
		revision = from + ".." + to
	}
	output, err := exec.CommandContext(ctx, "git", "rev-list", "--reverse", revision).Output()
	if err != nil {
		return nil, fmt.Errorf("list commits for %s: %w", revision, err)
	}
	lines := strings.Fields(string(output))
	if len(lines) == 0 {
		return nil, fmt.Errorf("no commits found for %s", revision)
	}
	return lines, nil
}

// pullRequestsForCommit 先使用 GitHub 的关联查询；squash merge 的最终 commit
// 可能没有关联结果，此时仅在提交标题带有 GitHub 生成的 (#N) 后缀、且 PR
// 的 merge_commit_sha 精确匹配时回溯 PR。这样既兼容 squash merge，也不会猜测来源。
func pullRequestsForCommit(ctx context.Context, client github.Client, repository, commit, title string) ([]model.GitHubPullRequest, error) {
	pulls, requestErr := client.PullRequestsForCommit(ctx, repository, commit)
	if requestErr != nil {
		pull, found, fallbackErr := squashPullRequest(ctx, client, repository, commit, title)
		if fallbackErr == nil && found {
			return []model.GitHubPullRequest{pull}, nil
		}
		if fallbackErr != nil {
			return nil, fmt.Errorf("lookup PRs for commit %s: %w (squash fallback: %v)", commit, requestErr, fallbackErr)
		}
		return nil, fmt.Errorf("lookup PRs for commit %s: %w", commit, requestErr)
	}
	if len(pulls) > 0 {
		return pulls, nil
	}
	pull, found, fallbackErr := squashPullRequest(ctx, client, repository, commit, title)
	if fallbackErr != nil {
		return nil, fmt.Errorf("resolve squash PR for commit %s: %w", commit, fallbackErr)
	}
	if found {
		return []model.GitHubPullRequest{pull}, nil
	}
	return pulls, nil
}

func squashPullRequest(ctx context.Context, client github.Client, repository, commit, title string) (model.GitHubPullRequest, bool, error) {
	matches := squashPullRequestPattern.FindStringSubmatch(title)
	if len(matches) != 2 {
		return model.GitHubPullRequest{}, false, nil
	}
	number, err := strconv.Atoi(matches[1])
	if err != nil {
		return model.GitHubPullRequest{}, true, fmt.Errorf("parse PR number from commit title %q: %w", title, err)
	}
	pull, err := client.PullRequest(ctx, repository, number)
	if err != nil {
		return model.GitHubPullRequest{}, true, fmt.Errorf("lookup PR #%d: %w", number, err)
	}
	if pull.MergedAt == nil {
		return model.GitHubPullRequest{}, true, fmt.Errorf("PR #%d is not merged", number)
	}
	if !strings.EqualFold(strings.TrimSpace(pull.MergeCommitSHA), commit) {
		return model.GitHubPullRequest{}, true, fmt.Errorf("PR #%d merge_commit_sha %q does not match %s", number, pull.MergeCommitSHA, commit)
	}
	return pull, true, nil
}

func gitCommitDetail(ctx context.Context, commit string) (title, author string, err error) {
	output, err := exec.CommandContext(ctx, "git", "show", "-s", "--format=%s%x00%an", commit).Output()
	if err != nil {
		return "", "", fmt.Errorf("read direct commit %s: %w", commit, err)
	}
	title, author, ok := strings.Cut(strings.TrimSuffix(string(output), "\n"), "\x00")
	if !ok || title == "" || author == "" {
		return "", "", fmt.Errorf("read direct commit %s: malformed git metadata", commit)
	}
	return title, author, nil
}
