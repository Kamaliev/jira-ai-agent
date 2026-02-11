package jira

type Issue struct {
	Key     string
	Summary string
	Status  string
}

type searchResponse struct {
	StartAt    int           `json:"startAt"`
	MaxResults int           `json:"maxResults"`
	Total      int           `json:"total"`
	Issues     []searchIssue `json:"issues"`
}

type searchIssue struct {
	Key    string      `json:"key"`
	Fields issueFields `json:"fields"`
}

type issueFields struct {
	Summary string       `json:"summary"`
	Status  *issueStatus `json:"status"`
}

type issueStatus struct {
	Name string `json:"name"`
}

type worklogPayload struct {
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	Comment          string `json:"comment"`
	Started          string `json:"started,omitempty"`
}

type myselfResponse struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
}

type worklogListResponse struct {
	Worklogs []worklogEntry `json:"worklogs"`
}

type worklogEntry struct {
	Author           worklogAuthor `json:"author"`
	TimeSpentSeconds int           `json:"timeSpentSeconds"`
	Started          string        `json:"started"`
}

type worklogAuthor struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
}
