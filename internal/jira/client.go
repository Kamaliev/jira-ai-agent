package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL  string
	apiToken string
	http     *http.Client
}

func NewClient(baseURL, email, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiToken: apiToken,
		http:     &http.Client{},
	}
}

func (c *Client) GetInProgressIssues(ctx context.Context) ([]Issue, error) {
	jql := `assignee = currentUser() AND status = "In Progress" ORDER BY updated DESC`

	u, err := url.Parse(c.baseURL + "/rest/api/2/search")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	q := u.Query()
	q.Set("jql", jql)
	q.Set("fields", "summary,status,assignee,description")
	q.Set("maxResults", "50")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira returned %d: %s", resp.StatusCode, string(body))
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	issues := make([]Issue, 0, len(sr.Issues))
	for _, si := range sr.Issues {
		status := ""
		if si.Fields.Status != nil {
			status = si.Fields.Status.Name
		}
		issues = append(issues, Issue{
			Key:     si.Key,
			Summary: si.Fields.Summary,
			Status:  status,
		})
	}

	return issues, nil
}

func (c *Client) LogWork(ctx context.Context, issueKey string, timeSpentSeconds int, description string) error {
	payload := worklogPayload{
		TimeSpentSeconds: timeSpentSeconds,
		Comment:          description,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	endpoint := c.baseURL + "/rest/api/2/issue/" + url.PathEscape(issueKey) + "/worklog"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("jira request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jira returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
