package jira

type Issue struct {
	Key     string
	Summary string
	Status  string
}

type searchResponse struct {
	Issues []searchIssue `json:"issues"`
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
}
