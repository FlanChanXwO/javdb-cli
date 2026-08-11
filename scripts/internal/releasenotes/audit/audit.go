// Package audit 负责读取 GitHub 元数据、校验 PR 声明并生成审计报告。
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
	"sort"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/document"
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
	requireClassified := flags.Bool("require-classified", false, "fail when a source has no usable release-note declaration")
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
	if err := WriteJSON(*output, report); err != nil {
		return err
	}
	if *requireClassified {
		for _, source := range report.Sources {
			if source.Issue != "" {
				return fmt.Errorf("audit has unresolved source %s: %s", source.URL, source.Issue)
			}
		}
	}
	return nil
}

// RunPullRequestValidate 执行 pr-validate 子命令。
func RunPullRequestValidate(arguments []string) error {
	flags := flag.NewFlagSet("pr-validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	eventPath := flags.String("event", "", "GitHub pull_request event JSON path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || *eventPath == "" {
		return errors.New("pr-validate requires --event")
	}
	return ValidatePullRequestEvent(*eventPath)
}

// ValidatePullRequestEvent 校验 GitHub pull_request 事件中的 release-note 声明。
func ValidatePullRequestEvent(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read pull request event: %w", err)
	}
	var event struct {
		PullRequest struct {
			Body string `json:"body"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		return fmt.Errorf("parse pull request event: %w", err)
	}
	_, err = document.ParseReleaseNoteDeclaration(event.PullRequest.Body)
	return err
}

// Collect 收集 commit、PR、release-note 和新贡献者信息。
func Collect(ctx context.Context, config Config) (model.AuditReport, error) {
	commits, err := gitRevisionList(ctx, config.From, config.To)
	if err != nil {
		return model.AuditReport{}, err
	}
	owner, _, _ := strings.Cut(config.Repository, "/")
	report := model.AuditReport{
		Repository:      config.Repository,
		From:            config.From,
		To:              config.To,
		Sources:         make([]model.AuditSource, 0),
		NewContributors: make([]model.NewContributor, 0),
	}
	seenPulls := make(map[int]struct{})
	seenContributors := make(map[string]struct{})
	firstMergedPulls := make(map[string]int)
	for _, commit := range commits {
		pulls, err := config.Client.PullRequestsForCommit(ctx, config.Repository, commit)
		if err != nil {
			return model.AuditReport{}, fmt.Errorf("lookup PRs for commit %s: %w", commit, err)
		}
		if len(pulls) == 0 {
			title, author, detailErr := gitCommitDetail(ctx, commit)
			if detailErr != nil {
				return model.AuditReport{}, detailErr
			}
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
			source := model.AuditSource{
				Kind:       "pull_request",
				URL:        pull.HTMLURL,
				PullNumber: pull.Number,
				Title:      pull.Title,
				Author:     pull.User.Login,
			}
			note, parseErr := document.ParseReleaseNoteDeclaration(pull.Body)
			if parseErr != nil {
				source.Issue = parseErr.Error()
			} else {
				source.Note = &note
			}
			report.Sources = append(report.Sources, source)
			if IsExternalContributor(pull, owner) {
				firstMergedPull, known := firstMergedPulls[pull.User.Login]
				if !known {
					// author_association 会随仓库历史变化，故以仓库中该作者
					// 最早合并的 PR 为准，并缓存结果避免同一作者重复查询。
					first, firstErr := config.Client.FirstMergedPullRequest(ctx, config.Repository, pull.User.Login)
					if firstErr != nil {
						return model.AuditReport{}, fmt.Errorf("lookup first merged PR for %q: %w", pull.User.Login, firstErr)
					}
					firstMergedPull = first.Number
					firstMergedPulls[pull.User.Login] = firstMergedPull
				}
				if firstMergedPull != pull.Number {
					continue
				}
				if _, exists := seenContributors[pull.User.Login]; !exists {
					seenContributors[pull.User.Login] = struct{}{}
					report.NewContributors = append(report.NewContributors, model.NewContributor{
						Login:      pull.User.Login,
						ProfileURL: pull.User.HTMLURL,
						PullNumber: pull.Number,
						PullURL:    pull.HTMLURL,
					})
				}
			}
		}
	}
	notes := make([]model.ReleaseNote, 0, len(report.Sources))
	for _, source := range report.Sources {
		if source.Note != nil {
			notes = append(notes, *source.Note)
		}
	}
	report.RecommendedVersionBump = document.RecommendedVersionBump(notes)
	sort.Slice(report.Sources, func(left, right int) bool { return report.Sources[left].URL < report.Sources[right].URL })
	sort.Slice(report.NewContributors, func(left, right int) bool {
		return report.NewContributors[left].Login < report.NewContributors[right].Login
	})
	return report, nil
}

// IsExternalContributor 判断 PR 作者是否是非仓库所有者、非机器人用户。
func IsExternalContributor(pull model.GitHubPullRequest, owner string) bool {
	if pull.User.Login == "" || pull.User.Login == owner {
		return false
	}
	return pull.User.Type != "Bot" && !strings.HasSuffix(strings.ToLower(pull.User.Login), "[bot]")
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
