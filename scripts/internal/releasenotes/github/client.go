// Package github 封装 release-note 流程使用的 GitHub REST 读写边界。
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/scripts/internal/releasenotes/model"
)

// Client 是 release-note 流程使用的 GitHub HTTP 客户端。
// Token 只用于请求头，不会被错误信息或报告序列化。
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New 构造 GitHub 客户端；nil HTTP client 保持由 requestJSON 使用默认客户端。
func New(baseURL, token string, httpClient *http.Client) Client {
	return Client{BaseURL: baseURL, Token: token, HTTPClient: httpClient}
}

// ValidRepository 判断 owner/name 形式的 GitHub 仓库标识。
func ValidRepository(repository string) bool {
	owner, name, ok := strings.Cut(repository, "/")
	return ok && owner != "" && name != "" && !strings.Contains(name, "/")
}

// PullRequestsForCommit 查询一个 commit 关联的 PR。
func (client Client) PullRequestsForCommit(ctx context.Context, repository, commit string) ([]model.GitHubPullRequest, error) {
	var pulls []model.GitHubPullRequest
	if err := client.GetJSON(ctx, "/repos/"+repository+"/commits/"+url.PathEscape(commit)+"/pulls", &pulls); err != nil {
		return nil, err
	}
	return pulls, nil
}

// PullRequest 读取指定 PR 的完整信息。
func (client Client) PullRequest(ctx context.Context, repository string, number int) (model.GitHubPullRequest, error) {
	var pull model.GitHubPullRequest
	if err := client.GetJSON(ctx, fmt.Sprintf("/repos/%s/pulls/%d", repository, number), &pull); err != nil {
		return model.GitHubPullRequest{}, err
	}
	return pull, nil
}

// FirstMergedPullRequest 找出某作者最早合并的 PR。
// Search API 不支持按 merged_at 排序，因此逐页取得完整候选详情后再比较时间。
func (client Client) FirstMergedPullRequest(ctx context.Context, repository, login string) (model.GitHubPullRequest, error) {
	pulls, err := client.MergedPullRequestsByAuthor(ctx, repository, login)
	if err != nil {
		return model.GitHubPullRequest{}, err
	}
	if len(pulls) == 0 {
		return model.GitHubPullRequest{}, fmt.Errorf("no merged pull request found")
	}
	first := pulls[0]
	for _, pull := range pulls[1:] {
		if pull.MergedAt.Before(*first.MergedAt) {
			first = pull
		}
	}
	return first, nil
}

// MergedPullRequestsByAuthor 读取某作者的全部已合并 PR。
func (client Client) MergedPullRequestsByAuthor(ctx context.Context, repository, login string) ([]model.GitHubPullRequest, error) {
	query := url.Values{}
	query.Set("q", "repo:"+repository+" type:pr author:"+login+" is:merged")
	query.Set("per_page", "100")
	pulls := make([]model.GitHubPullRequest, 0)
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		var result model.GitHubPullRequestSearchResult
		if err := client.GetJSON(ctx, "/search/issues?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		for _, summary := range result.Items {
			pull, err := client.PullRequest(ctx, repository, summary.Number)
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

// GetJSON 执行 GitHub REST GET 请求。
func (client Client) GetJSON(ctx context.Context, path string, destination any) error {
	return client.RequestJSON(ctx, http.MethodGet, path, nil, destination)
}

// RequestJSON 执行 GitHub REST JSON 请求。
func (client Client) RequestJSON(ctx context.Context, method, path string, input, destination any) error {
	baseURL := strings.TrimRight(client.BaseURL, "/")
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
	if client.Token != "" {
		request.Header.Set("Authorization", "Bearer "+client.Token)
	}
	httpClient := client.HTTPClient
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
			return &HTTPError{Path: path, StatusCode: response.StatusCode}
		}
		return &HTTPError{Path: path, StatusCode: response.StatusCode, Body: strings.TrimSpace(string(responseBody))}
	}
	if destination == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(destination)
}

// HTTPError 保留 GitHub 非 2xx 响应的路径、状态码和正文摘要。
type HTTPError struct {
	Path       string
	StatusCode int
	Body       string
}

func (err *HTTPError) Error() string {
	if err.Body == "" {
		return fmt.Sprintf("GitHub API %s: HTTP %d", err.Path, err.StatusCode)
	}
	return fmt.Sprintf("GitHub API %s: HTTP %d: %s", err.Path, err.StatusCode, err.Body)
}
