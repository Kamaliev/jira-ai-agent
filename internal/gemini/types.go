package gemini

type WorkLog struct {
	IssueKey    string `json:"issue_key"`
	TimeSpent   string `json:"time_spent"`
	Description string `json:"description"`
}

type InterviewResult struct {
	WorkLogs      []WorkLog `json:"work_logs"`
	ReadyToSubmit bool      `json:"ready_to_submit"`
}

type ParsedWorkLog struct {
	IssueKey    string
	TimeSeconds int
	Description string
	Summary     string
}
