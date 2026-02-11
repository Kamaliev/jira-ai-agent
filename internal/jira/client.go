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
	"time"
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

func (c *Client) GetMyIssues(ctx context.Context) ([]Issue, error) {
	jql := `assignee = currentUser() AND status != "Done" AND issuetype not in (Story, Epic) ORDER BY updated DESC`
	return c.searchIssues(ctx, jql)
}

func (c *Client) GetAllIssues(ctx context.Context) ([]Issue, error) {
	jql := `status != "Done" AND issuetype not in (Story, Epic) ORDER BY updated DESC`
	return c.searchIssues(ctx, jql)
}

func (c *Client) searchIssues(ctx context.Context, jql string) ([]Issue, error) {
	var issues []Issue
	startAt := 0
	const pageSize = 100

	for {
		u, err := url.Parse(c.baseURL + "/rest/api/2/search")
		if err != nil {
			return nil, fmt.Errorf("parse URL: %w", err)
		}

		q := u.Query()
		q.Set("jql", jql)
		q.Set("fields", "summary,status")
		q.Set("maxResults", fmt.Sprintf("%d", pageSize))
		q.Set("startAt", fmt.Sprintf("%d", startAt))
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

		startAt += len(sr.Issues)
		if startAt >= sr.Total {
			break
		}
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

func (c *Client) GetTodayLoggedSeconds(ctx context.Context) (int, error) {
	accountID, err := c.getMyAccountID(ctx)
	if err != nil {
		return 0, fmt.Errorf("get current user: %w", err)
	}

	jql := `worklogDate >= startOfDay() AND worklogAuthor = currentUser()`
	issues, err := c.searchIssues(ctx, jql)
	if err != nil {
		return 0, fmt.Errorf("search worklogs: %w", err)
	}

	today := time.Now().Format("2006-01-02")
	totalSeconds := 0

	for _, issue := range issues {
		worklogs, err := c.getIssueWorklogs(ctx, issue.Key)
		if err != nil {
			continue
		}
		for _, wl := range worklogs {
			if (wl.Author.AccountID == accountID || wl.Author.Name == accountID) && strings.HasPrefix(wl.Started, today) {
				totalSeconds += wl.TimeSpentSeconds
			}
		}
	}

	return totalSeconds, nil
}

func (c *Client) getMyAccountID(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/rest/api/2/myself", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("jira returned %d: %s", resp.StatusCode, string(body))
	}

	var me myselfResponse
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return "", err
	}

	if me.AccountID != "" {
		return me.AccountID, nil
	}
	return me.Name, nil
}

func (c *Client) getIssueWorklogs(ctx context.Context, issueKey string) ([]worklogEntry, error) {
	endpoint := c.baseURL + "/rest/api/2/issue/" + url.PathEscape(issueKey) + "/worklog"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jira returned %d", resp.StatusCode)
	}

	var wlResp worklogListResponse
	if err := json.NewDecoder(resp.Body).Decode(&wlResp); err != nil {
		return nil, err
	}

	return wlResp.Worklogs, nil
}
