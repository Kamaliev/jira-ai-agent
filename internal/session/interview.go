package session

import (
	"context"
	"time"

	"go-secretary/internal/gemini"
	"go-secretary/internal/jira"
	"go-secretary/internal/ui"

	"github.com/pterm/pterm"
)

type Runner struct {
	jira   *jira.Client
	gemini *gemini.Assistant
}

func NewRunner(jiraClient *jira.Client, geminiAssistant *gemini.Assistant) *Runner {
	return &Runner{
		jira:   jiraClient,
		gemini: geminiAssistant,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	ui.PrintWelcome()

	// Fetch issues with spinner
	spinner, _ := pterm.DefaultSpinner.Start("Получаю твои задачи из Jira...")
	issues, err := r.jira.GetInProgressIssues(ctx)
	spinner.Stop()
	if err != nil {
		ui.PrintError("Ошибка при получении задач из Jira: " + err.Error())
		return err
	}

	if len(issues) == 0 {
		ui.PrintNoIssues()
		return nil
	}

	ui.PrintIssuesTable(issues)
	ui.PrintStatus("Сейчас AI-ассистент проведёт с тобой интервью...")
	time.Sleep(1 * time.Second)

	// Start AI interview
	response, err := r.gemini.StartInterview(ctx, issues)
	if err != nil {
		ui.PrintError("Ошибка при общении с Gemini: " + err.Error())
		return err
	}
	ui.PrintTypewriter(response)

	// Conversation loop
	const maxTurns = 20
	for turn := 0; turn < maxTurns; turn++ {
		// Check if work logs are ready
		workLogs := r.gemini.ExtractWorkLogs(response)
		if workLogs != nil {
			return r.handleSubmission(ctx, workLogs)
		}

		// Get user input
		userInput := ui.ReadInput("Ты: ")
		if userInput == "" {
			continue
		}
		if ui.IsExitCommand(userInput) {
			pterm.Println()
			ui.PrintStatus("Диалог прерван. До встречи!")
			return nil
		}

		// Send to Gemini
		thinkSpinner, _ := pterm.DefaultSpinner.
			WithRemoveWhenDone(true).
			Start("Gemini думает...")
		response, err = r.gemini.SendMessage(ctx, userInput)
		thinkSpinner.Stop()
		if err != nil {
			ui.PrintError("Ошибка при общении с Gemini: " + err.Error())
			return err
		}
		ui.PrintTypewriter(response)
	}

	// Final extraction attempt
	workLogs := r.gemini.ExtractWorkLogs(response)
	if workLogs == nil {
		ui.PrintStatus("Собираю данные...")
		response, err = r.gemini.SendMessage(ctx, "Отлично! Теперь, пожалуйста, верни мне все собранные данные в JSON формате.")
		if err == nil {
			workLogs = r.gemini.ExtractWorkLogs(response)
		}
	}

	if workLogs == nil {
		ui.PrintNoData()
		return nil
	}

	return r.handleSubmission(ctx, workLogs)
}

func (r *Runner) handleSubmission(ctx context.Context, workLogs []gemini.ParsedWorkLog) error {
	ui.PrintSummary(workLogs)

	if !ui.ConfirmYesNo("Отправить эти данные в Jira?") {
		ui.PrintCancelled()
		return nil
	}

	pterm.Println()
	for _, log := range workLogs {
		spinner, _ := pterm.DefaultSpinner.Start("Логирую " + log.IssueKey + "...")
		err := r.jira.LogWork(ctx, log.IssueKey, log.TimeSeconds, log.Description)
		spinner.Stop()
		ui.PrintLogResult(log.IssueKey, err == nil)
		if err != nil {
			ui.PrintError("  " + err.Error())
		}
		time.Sleep(300 * time.Millisecond)
	}

	ui.PrintFarewell()
	return nil
}
