// 审计负责读取 GitHub 元数据、校验 PR 声明并序列化审计报告。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func runAudit(arguments []string) error {
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
	if !validRepository(*repository) || *to == "" {
		return errors.New("audit requires --repo owner/name and --to")
	}
	report, err := collectAudit(context.Background(), auditConfig{
		repository: *repository,
		from:       *from,
		to:         *to,
		client: githubClient{
			baseURL: *apiBase,
			token:   os.Getenv(*tokenEnv),
			client:  http.DefaultClient,
		},
	})
	if err != nil {
		return err
	}
	if err := writeJSON(*output, report); err != nil {
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

func runPullRequestValidate(arguments []string) error {
	flags := flag.NewFlagSet("pr-validate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	eventPath := flags.String("event", "", "GitHub pull_request event JSON path")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 || *eventPath == "" {
		return errors.New("pr-validate requires --event")
	}
	return validatePullRequestEvent(*eventPath)
}

func validatePullRequestEvent(path string) error {
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
	_, err = parseReleaseNoteDeclaration(event.PullRequest.Body)
	return err
}

type auditConfig struct {
	repository string
	from       string
	to         string
	client     githubClient
}

func collectAudit(ctx context.Context, config auditConfig) (auditReport, error) {
	commits, err := gitRevisionList(ctx, config.from, config.to)
	if err != nil {
		return auditReport{}, err
	}
	owner, _, _ := strings.Cut(config.repository, "/")
	report := auditReport{
		Repository:      config.repository,
		From:            config.from,
		To:              config.to,
		Sources:         make([]auditSource, 0),
		NewContributors: make([]newContributor, 0),
	}
	seenPulls := make(map[int]struct{})
	seenContributors := make(map[string]struct{})
	firstMergedPulls := make(map[string]int)
	for _, commit := range commits {
		pulls, err := config.client.pullRequestsForCommit(ctx, config.repository, commit)
		if err != nil {
			return auditReport{}, fmt.Errorf("lookup PRs for commit %s: %w", commit, err)
		}
		if len(pulls) == 0 {
			title, author, detailErr := gitCommitDetail(ctx, commit)
			if detailErr != nil {
				return auditReport{}, detailErr
			}
			report.Sources = append(report.Sources, auditSource{
				Kind:   "commit",
				URL:    "https://github.com/" + config.repository + "/commit/" + commit,
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
			pull, err := config.client.pullRequest(ctx, config.repository, summary.Number)
			if err != nil {
				return auditReport{}, fmt.Errorf("lookup PR #%d: %w", summary.Number, err)
			}
			source := auditSource{
				Kind:       "pull_request",
				URL:        pull.HTMLURL,
				PullNumber: pull.Number,
				Title:      pull.Title,
				Author:     pull.User.Login,
			}
			note, parseErr := parseReleaseNoteDeclaration(pull.Body)
			if parseErr != nil {
				source.Issue = parseErr.Error()
			} else {
				source.Note = &note
			}
			report.Sources = append(report.Sources, source)
			if isExternalContributor(pull, owner) {
				firstMergedPull, known := firstMergedPulls[pull.User.Login]
				if !known {
					// author_association 会随仓库历史变化，故以仓库中该作者
					// 最早合并的 PR 为准，并缓存结果避免同一作者重复查询。
					first, firstErr := config.client.firstMergedPullRequest(ctx, config.repository, pull.User.Login)
					if firstErr != nil {
						return auditReport{}, fmt.Errorf("lookup first merged PR for %q: %w", pull.User.Login, firstErr)
					}
					firstMergedPull = first.Number
					firstMergedPulls[pull.User.Login] = firstMergedPull
				}
				if firstMergedPull != pull.Number {
					continue
				}
				if _, exists := seenContributors[pull.User.Login]; !exists {
					seenContributors[pull.User.Login] = struct{}{}
					report.NewContributors = append(report.NewContributors, newContributor{
						Login:      pull.User.Login,
						ProfileURL: pull.User.HTMLURL,
						PullNumber: pull.Number,
						PullURL:    pull.HTMLURL,
					})
				}
			}
		}
	}
	notes := make([]releaseNote, 0, len(report.Sources))
	for _, source := range report.Sources {
		if source.Note != nil {
			notes = append(notes, *source.Note)
		}
	}
	report.RecommendedVersionBump = recommendedVersionBump(notes)
	sort.Slice(report.Sources, func(left, right int) bool { return report.Sources[left].URL < report.Sources[right].URL })
	sort.Slice(report.NewContributors, func(left, right int) bool {
		return report.NewContributors[left].Login < report.NewContributors[right].Login
	})
	return report, nil
}

func (client githubClient) pullRequestsForCommit(ctx context.Context, repository, commit string) ([]githubPullRequest, error) {
	var pulls []githubPullRequest
	if err := client.getJSON(ctx, "/repos/"+repository+"/commits/"+url.PathEscape(commit)+"/pulls", &pulls); err != nil {
		return nil, err
	}
	return pulls, nil
}

func (client githubClient) pullRequest(ctx context.Context, repository string, number int) (githubPullRequest, error) {
	var pull githubPullRequest
	if err := client.getJSON(ctx, fmt.Sprintf("/repos/%s/pulls/%d", repository, number), &pull); err != nil {
		return githubPullRequest{}, err
	}
	return pull, nil
}

// firstMergedPullRequest 使用仓库的完整 PR 历史，而不是当前 author_association，
// 判定某个作者最早合并的 PR。Search API 不支持按 merged_at 排序，因此必须逐页取得
// 全部候选 PR 的详情后再比较合并时间，不能把创建时间误当作首次贡献时间。
func (client githubClient) firstMergedPullRequest(ctx context.Context, repository, login string) (githubPullRequest, error) {
	pulls, err := client.mergedPullRequestsByAuthor(ctx, repository, login)
	if err != nil {
		return githubPullRequest{}, err
	}
	if len(pulls) == 0 {
		return githubPullRequest{}, fmt.Errorf("no merged pull request found")
	}
	first := pulls[0]
	for _, pull := range pulls[1:] {
		if pull.MergedAt.Before(*first.MergedAt) {
			first = pull
		}
	}
	return first, nil
}

func (client githubClient) mergedPullRequestsByAuthor(ctx context.Context, repository, login string) ([]githubPullRequest, error) {
	query := url.Values{}
	query.Set("q", "repo:"+repository+" type:pr author:"+login+" is:merged")
	query.Set("per_page", "100")
	pulls := make([]githubPullRequest, 0)
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		var result githubPullRequestSearchResult
		if err := client.getJSON(ctx, "/search/issues?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		for _, summary := range result.Items {
			pull, err := client.pullRequest(ctx, repository, summary.Number)
			if err != nil {
				return nil, err
			}
			if pull.MergedAt != nil {
				pulls = append(pulls, pull)
			}
		}
		if len(result.Items) < 100 {
			return pulls, nil
		}
	}
}

func (client githubClient) getJSON(ctx context.Context, path string, destination any) error {
	return client.requestJSON(ctx, http.MethodGet, path, nil, destination)
}

type githubHTTPError struct {
	Path       string
	StatusCode int
	Body       string
}

func (err *githubHTTPError) Error() string {
	if err.Body == "" {
		return fmt.Sprintf("GitHub API %s: HTTP %d", err.Path, err.StatusCode)
	}
	return fmt.Sprintf("GitHub API %s: HTTP %d: %s", err.Path, err.StatusCode, err.Body)
}

func (client githubClient) requestJSON(ctx context.Context, method, path string, input, destination any) error {
	baseURL := strings.TrimRight(client.baseURL, "/")
	if baseURL == "" {
		return errors.New("GitHub API base URL is required")
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("encode GitHub API request: %w", err)
		}
		body = strings.NewReader(string(encoded))
	}
	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}
	httpClient := client.client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return &githubHTTPError{Path: path, StatusCode: response.StatusCode}
		}
		return &githubHTTPError{Path: path, StatusCode: response.StatusCode, Body: strings.TrimSpace(string(responseBody))}
	}
	if destination == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(destination)
}

func isExternalContributor(pull githubPullRequest, owner string) bool {
	if pull.User.Login == "" || pull.User.Login == owner {
		return false
	}
	return pull.User.Type != "Bot" && !strings.HasSuffix(strings.ToLower(pull.User.Login), "[bot]")
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

func validRepository(repository string) bool {
	owner, name, ok := strings.Cut(repository, "/")
	return ok && owner != "" && name != "" && !strings.Contains(name, "/")
}

func writeJSON(path string, value any) error {
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
