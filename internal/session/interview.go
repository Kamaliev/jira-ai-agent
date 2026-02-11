package session

import (
	"context"
	"fmt"
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

	// Fetch user's own issues to display as a reminder
	spinner, _ := pterm.DefaultSpinner.Start("Получаю твои задачи из Jira...")
	myIssues, err := r.jira.GetMyIssues(ctx)
	spinner.Stop()
	if err != nil {
		ui.PrintError("Ошибка при получении задач из Jira: " + err.Error())
		return err
	}

	if len(myIssues) > 0 {
		ui.PrintIssuesTable(myIssues)
	}

	// Fetch all issues for Gemini context
	spinner, _ = pterm.DefaultSpinner.Start("Загружаю все задачи проекта...")
	allIssues, err := r.jira.GetAllIssues(ctx)
	spinner.Stop()
	if err != nil {
		ui.PrintError("Ошибка при получении задач из Jira: " + err.Error())
		return err
	}

	if len(allIssues) == 0 {
		ui.PrintNoIssues()
		return nil
	}

	// Check today's already logged time
	spinner, _ = pterm.DefaultSpinner.Start("Проверяю ворклоги за сегодня...")
	loggedSeconds, err := r.jira.GetTodayLoggedSeconds(ctx)
	spinner.Stop()
	if err != nil {
		ui.PrintError("Ошибка при получении ворклогов: " + err.Error())
		return err
	}

	if loggedSeconds > 0 {
		h := loggedSeconds / 3600
		m := (loggedSeconds % 3600) / 60
		ui.PrintStatus(fmt.Sprintf("Сегодня уже залогировано: %dh %dm", h, m))
	}

	ui.PrintStatus("Расскажи AI-ассистенту, чем ты сегодня занимался...")
	time.Sleep(1 * time.Second)

	// Start AI conversation
	response, err := r.gemini.StartConversation(ctx, allIssues, loggedSeconds)
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
		response, err = r.gemini.SendMessage(ctx, "Пожалуйста, подведи итог и верни все собранные данные в JSON формате.")
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
