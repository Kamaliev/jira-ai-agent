package ui

import (
	"fmt"

	"go-secretary/internal/gemini"
	"go-secretary/internal/jira"

	"github.com/pterm/pterm"
)

func PrintWelcome() {
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println("Jira Tempo AI Agent")
	pterm.Println(pterm.Gray("Ваш AI-ассистент на базе Google Gemini"))
	pterm.Println()
}

func PrintIssuesTable(issues []jira.Issue) {
	pterm.Success.Printfln("Найдено задач: %d", len(issues))
	pterm.Println()

	tableData := pterm.TableData{
		{"#", "Ключ", "Название"},
	}
	for i, issue := range issues {
		tableData = append(tableData, []string{
			fmt.Sprintf("%d", i+1),
			issue.Key,
			issue.Summary,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()
	pterm.Println()
}

func PrintSummary(logs []gemini.ParsedWorkLog) {
	pterm.Println()
	pterm.DefaultSection.WithStyle(pterm.NewStyle(pterm.FgCyan, pterm.Bold)).Println("Итоговая сводка")

	tableData := pterm.TableData{
		{"Задача", "Время", "Описание"},
	}

	totalSeconds := 0
	for _, log := range logs {
		hours := float64(log.TimeSeconds) / 3600.0
		totalSeconds += log.TimeSeconds
		tableData = append(tableData, []string{
			pterm.FgCyan.Sprint(log.IssueKey),
			pterm.FgYellow.Sprintf("%.1fч", hours),
			log.Description,
		})
	}

	totalHours := float64(totalSeconds) / 3600.0
	tableData = append(tableData, []string{
		pterm.Bold.Sprint("ИТОГО"),
		pterm.Bold.Sprint(pterm.FgYellow.Sprintf("%.1fч", totalHours)),
		"",
	})

	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()
	pterm.Println()
}

func PrintLogResult(issueKey string, success bool) {
	if success {
		pterm.Success.Printfln("%s", issueKey)
	} else {
		pterm.Error.Printfln("%s", issueKey)
	}
}

func PrintNoIssues() {
	pterm.Warning.Println("Не найдено задач в статусе 'In Progress'")
	pterm.Println(pterm.Gray("Проверь, что у тебя есть задачи в работе в Jira."))
}

func PrintNoData() {
	pterm.Warning.Println("Не удалось собрать данные.")
	pterm.Println(pterm.Gray("Попробуй начать заново."))
}

func PrintCancelled() {
	pterm.Warning.Println("Отменено.")
}

func PrintFarewell() {
	pterm.Println()
	pterm.Println(pterm.Gray("Спасибо! До встречи!"))
	pterm.Println()
}

func PrintError(msg string) {
	pterm.Println(pterm.Gray("⚠ " + msg))
}

func PrintStatus(msg string) {
	pterm.Println(pterm.Gray(msg))
}

func PrintPeriodStatus(days []DayStatus) {
	tableData := pterm.TableData{
		{"Дата", "День недели", "Залогировано", "Статус"},
	}
	for _, d := range days {
		h := d.LoggedSeconds / 3600
		m := (d.LoggedSeconds % 3600) / 60
		logged := fmt.Sprintf("%dh %dm", h, m)

		status := pterm.FgRed.Sprint("Не заполнено")
		if d.Filled {
			status = pterm.FgGreen.Sprint("OK")
		}

		tableData = append(tableData, []string{
			d.Date,
			d.Weekday,
			logged,
			status,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithBoxed().WithData(tableData).Render()
	pterm.Println()
}

func PrintDayHeader(date string) {
	pterm.Println()
	pterm.DefaultHeader.WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Printfln("=== %s ===", date)
	pterm.Println()
}
