package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// Issue represents a GitHub issue.
type Issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
}

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	Number int    `json:"number"`
	NodeID string `json:"node_id"`
	Merged bool   `json:"merged"`
}

// Client calls the GitHub REST API.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// NewClient returns a Client authenticated via GITHUB_TOKEN if token is empty.
func NewClient(token string) *Client {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	return &Client{
		token:   token,
		baseURL: "https://api.github.com",
		http:    &http.Client{},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body string) (*http.Response, error) {
	url := c.baseURL + path
	var reqBody *strings.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	} else {
		reqBody = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("github API %s %s: status %d", method, path, resp.StatusCode)
	}
	return resp, nil
}

// GetReadyIssues returns all open issues labeled ready-for-agent in repo (owner/name).
func (c *Client) GetReadyIssues(ctx context.Context, repo string) ([]Issue, error) {
	path := fmt.Sprintf("/repos/%s/issues?state=open&labels=ready-for-agent", repo)
	resp, err := c.do(ctx, http.MethodGet, path, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}
	return issues, nil
}

// GetIssue fetches a single issue by number.
func (c *Client) GetIssue(ctx context.Context, repo string, number int) (Issue, error) {
	path := fmt.Sprintf("/repos/%s/issues/%d", repo, number)
	resp, err := c.do(ctx, http.MethodGet, path, "")
	if err != nil {
		return Issue{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return Issue{}, err
	}
	return issue, nil
}

// IsIssueClosed returns true if the issue state is "closed".
func (c *Client) IsIssueClosed(ctx context.Context, repo string, number int) (bool, error) {
	issue, err := c.GetIssue(ctx, repo, number)
	if err != nil {
		return false, err
	}
	return issue.State == "closed", nil
}

// GetPRByBranch returns the open PR for head branch in repo. Returns ok=false if none found.
func (c *Client) GetPRByBranch(ctx context.Context, repo, branch string) (PullRequest, bool, error) {
	owner := strings.SplitN(repo, "/", 2)[0]
	path := fmt.Sprintf("/repos/%s/pulls?state=open&head=%s:%s", repo, owner, branch)
	resp, err := c.do(ctx, http.MethodGet, path, "")
	if err != nil {
		return PullRequest{}, false, err
	}
	defer func() { _ = resp.Body.Close() }()
	var prs []PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return PullRequest{}, false, err
	}
	if len(prs) == 0 {
		return PullRequest{}, false, nil
	}
	return prs[0], true, nil
}

// GetPR returns a single PR by number, including merged status.
func (c *Client) GetPR(ctx context.Context, repo string, number int) (PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/pulls/%d", repo, number)
	resp, err := c.do(ctx, http.MethodGet, path, "")
	if err != nil {
		return PullRequest{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return PullRequest{}, err
	}
	return pr, nil
}

// EnableAutoMerge enables squash auto-merge on the given PR via GitHub GraphQL.
func (c *Client) EnableAutoMerge(ctx context.Context, repo string, prNumber int) error {
	pr, err := c.GetPR(ctx, repo, prNumber)
	if err != nil {
		return fmt.Errorf("get PR node_id: %w", err)
	}
	query := fmt.Sprintf(
		`{"query":"mutation { enablePullRequestAutoMerge(input: {pullRequestId: \"%s\", mergeMethod: SQUASH}) { pullRequest { id } } }"}`,
		pr.NodeID,
	)
	resp, err := c.do(ctx, http.MethodPost, "/graphql", query)
	if err != nil {
		return fmt.Errorf("enable auto-merge: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// UpdateLabel adds add label and removes remove label atomically.
func (c *Client) UpdateLabel(ctx context.Context, repo string, number int, add, remove string) error {
	addPath := fmt.Sprintf("/repos/%s/issues/%d/labels", repo, number)
	addBody := fmt.Sprintf(`{"labels":["%s"]}`, add)
	resp, err := c.do(ctx, http.MethodPost, addPath, addBody)
	if err != nil {
		return fmt.Errorf("add label: %w", err)
	}
	_ = resp.Body.Close()

	removePath := fmt.Sprintf("/repos/%s/issues/%d/labels/%s", repo, number, remove)
	resp, err = c.do(ctx, http.MethodDelete, removePath, "")
	if err != nil {
		return fmt.Errorf("remove label: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}
