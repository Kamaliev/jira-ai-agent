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

	return r.runConversation(ctx, allIssues, loggedSeconds, "")
}

func (r *Runner) RunPeriod(ctx context.Context) error {
	ui.PrintWelcome()

	// Ask for date range
	startDate, endDate, err := ui.ReadDateRange()
	if err != nil {
		ui.PrintError("Ошибка при вводе дат: " + err.Error())
		return err
	}

	// Get logged seconds for the period
	spinner, _ := pterm.DefaultSpinner.Start("Проверяю ворклоги за период...")
	loggedByDay, err := r.jira.GetLoggedSecondsForDateRange(ctx, startDate, endDate)
	spinner.Stop()
	if err != nil {
		ui.PrintError("Ошибка при получении ворклогов: " + err.Error())
		return err
	}

	// Calculate workdays and their status
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	var days []ui.DayStatus
	var unfilledDays []ui.DayStatus
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		wd := d.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			continue
		}

		dateStr := d.Format("2006-01-02")
		logged := loggedByDay[dateStr]
		filled := logged >= 8*3600

		ds := ui.DayStatus{
			Date:          dateStr,
			Weekday:       russianWeekday(wd),
			LoggedSeconds: logged,
			Filled:        filled,
		}
		days = append(days, ds)
		if !filled {
			unfilledDays = append(unfilledDays, ds)
		}
	}

	// Show period status table
	ui.PrintPeriodStatus(days)

	if len(unfilledDays) == 0 {
		ui.PrintStatus("Все рабочие дни за период заполнены!")
		return nil
	}

	pterm.Success.Printfln("Незаполненных дней: %d", len(unfilledDays))
	pterm.Println()

	// Load issues once for all days
	spinner, _ = pterm.DefaultSpinner.Start("Получаю твои задачи из Jira...")
	myIssues, err := r.jira.GetMyIssues(ctx)
	spinner.Stop()
	if err != nil {
		ui.PrintError("Ошибка при получении задач из Jira: " + err.Error())
		return err
	}

	if len(myIssues) > 0 {
		ui.PrintIssuesTable(myIssues)
	}

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

	// Process each unfilled day
	for _, day := range unfilledDays {
		ui.PrintDayHeader(fmt.Sprintf("%s, %s", day.Weekday, day.Date))

		if day.LoggedSeconds > 0 {
			h := day.LoggedSeconds / 3600
			m := (day.LoggedSeconds % 3600) / 60
			ui.PrintStatus(fmt.Sprintf("Уже залогировано за этот день: %dh %dm", h, m))
		}

		ui.PrintStatus(fmt.Sprintf("Расскажи AI-ассистенту, чем ты занимался %s...", day.Date))
		time.Sleep(500 * time.Millisecond)

		if err := r.runConversation(ctx, allIssues, day.LoggedSeconds, day.Date); err != nil {
			return err
		}
	}

	pterm.Println()
	ui.PrintStatus("Все незаполненные дни обработаны!")
	return nil
}

// runConversation runs the AI conversation loop for a single day.
// If date is empty, worklogs are logged with current time (today mode).
// If date is set, worklogs are logged with that specific date.
func (r *Runner) runConversation(ctx context.Context, allIssues []jira.Issue, loggedSeconds int, date string) error {
	response, err := r.gemini.StartConversation(ctx, allIssues, loggedSeconds, date)
	if err != nil {
		ui.PrintError("Ошибка при общении с Gemini: " + err.Error())
		return err
	}
	ui.PrintTypewriter(response)

	const maxTurns = 20
	for turn := 0; turn < maxTurns; turn++ {
		workLogs := r.gemini.ExtractWorkLogs(response)
		if workLogs != nil {
			return r.handleSubmissionForDate(ctx, workLogs, date)
		}

		userInput := ui.ReadInput("Ты: ")
		if userInput == "" {
			continue
		}
		if ui.IsExitCommand(userInput) {
			pterm.Println()
			ui.PrintStatus("Диалог прерван. До встречи!")
			return nil
		}

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

	return r.handleSubmissionForDate(ctx, workLogs, date)
}

func (r *Runner) handleSubmissionForDate(ctx context.Context, workLogs []gemini.ParsedWorkLog, date string) error {
	ui.PrintSummary(workLogs)

	if !ui.ConfirmYesNo("Отправить эти данные в Jira?") {
		ui.PrintCancelled()
		return nil
	}

	var started time.Time
	if date != "" {
		started, _ = time.Parse("2006-01-02", date)
	}

	pterm.Println()
	for _, log := range workLogs {
		spinner, _ := pterm.DefaultSpinner.Start("Логирую " + log.IssueKey + "...")
		err := r.jira.LogWork(ctx, log.IssueKey, log.TimeSeconds, log.Description, started)
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

func russianWeekday(wd time.Weekday) string {
	switch wd {
	case time.Monday:
		return "Понедельник"
	case time.Tuesday:
		return "Вторник"
	case time.Wednesday:
		return "Среда"
	case time.Thursday:
		return "Четверг"
	case time.Friday:
		return "Пятница"
	case time.Saturday:
		return "Суббота"
	case time.Sunday:
		return "Воскресенье"
	}
	return ""
}
