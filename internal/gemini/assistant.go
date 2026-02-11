package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-secretary/internal/jira"
	"go-secretary/internal/timeparse"

	"github.com/pterm/pterm"
	"google.golang.org/genai"
)

type Assistant struct {
	client *genai.Client
	chat   *genai.Chat
	model  string
}

func NewAssistant(ctx context.Context, apiKey, model string) (*Assistant, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}
	return &Assistant{client: client, model: model}, nil
}

func (a *Assistant) StartConversation(ctx context.Context, issues []jira.Issue, loggedSeconds int, date string) (string, error) {
	systemPrompt := buildSystemPrompt(issues, loggedSeconds, date)

	var err error
	a.chat, err = a.client.Chats.Create(ctx, "models/"+a.model, &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(systemPrompt, genai.RoleUser),
	}, nil)
	if err != nil {
		return "", fmt.Errorf("create chat: %w", err)
	}

	resp, err := a.sendWithRetry(ctx, "Привет! Готов начать.")
	if err != nil {
		return "", fmt.Errorf("start interview: %w", err)
	}

	return extractText(resp), nil
}

func (a *Assistant) SendMessage(ctx context.Context, message string) (string, error) {
	if a.chat == nil {
		return "", fmt.Errorf("chat not initialized")
	}

	resp, err := a.sendWithRetry(ctx, message)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	return extractText(resp), nil
}

func (a *Assistant) sendWithRetry(ctx context.Context, text string) (*genai.GenerateContentResponse, error) {
	const maxRetries = 3
	for attempt := range maxRetries {
		resp, err := a.chat.SendMessage(ctx, genai.Part{Text: text})
		if err == nil {
			return resp, nil
		}
		errMsg := err.Error()
		if !strings.Contains(errMsg, "429") && !strings.Contains(errMsg, "RESOURCE_EXHAUSTED") &&
			!strings.Contains(errMsg, "503") && !strings.Contains(errMsg, "UNAVAILABLE") {
			return nil, err
		}
		wait := time.Duration(30*(attempt+1)) * time.Second
		pterm.Println(pterm.Gray(fmt.Sprintf("⚠ %s — повтор через %v...", errMsg, wait)))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return a.chat.SendMessage(ctx, genai.Part{Text: text})
}

func (a *Assistant) ExtractWorkLogs(text string) []ParsedWorkLog {
	jsonStr := ""

	if idx := strings.Index(text, "```json"); idx >= 0 {
		rest := text[idx+7:]
		if end := strings.Index(rest, "```"); end >= 0 {
			jsonStr = strings.TrimSpace(rest[:end])
		}
	} else if strings.Contains(text, "```") && strings.Contains(text, "{") {
		parts := strings.SplitN(text, "```", 3)
		if len(parts) >= 2 {
			jsonStr = strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(text, "{") && strings.Contains(text, "}") {
		start := strings.Index(text, "{")
		end := strings.LastIndex(text, "}")
		if start >= 0 && end > start {
			jsonStr = text[start : end+1]
		}
	}

	if jsonStr == "" {
		return nil
	}

	var result InterviewResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil
	}

	if !result.ReadyToSubmit {
		return nil
	}

	var logs []ParsedWorkLog
	for _, wl := range result.WorkLogs {
		seconds := timeparse.Parse(wl.TimeSpent)
		if seconds > 0 {
			logs = append(logs, ParsedWorkLog{
				IssueKey:    wl.IssueKey,
				TimeSeconds: seconds,
				Description: wl.Description,
			})
		}
	}

	if len(logs) == 0 {
		return nil
	}

	return logs
}

func (a *Assistant) SetModel(model string) {
	a.model = model
	a.chat = nil
}

func (a *Assistant) Model() string {
	return a.model
}

func (a *Assistant) Close() {
	// genai client doesn't require explicit close
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

func buildSystemPrompt(issues []jira.Issue, loggedSeconds int, date string) string {
	var sb strings.Builder
	for _, issue := range issues {
		fmt.Fprintf(&sb, "- %s: %s\n", issue.Key, issue.Summary)
	}

	dayLabel := "сегодня"
	dayLabelAccusative := "чем ты сегодня занимался"
	if date != "" {
		dayLabel = "за " + date
		dayLabelAccusative = "чем ты занимался " + date
	}

	remainingSeconds := 8*3600 - loggedSeconds
	if remainingSeconds < 0 {
		remainingSeconds = 0
	}

	timeInfo := ""
	if loggedSeconds > 0 {
		timeInfo = fmt.Sprintf("\nУЖЕ ЗАЛОГИРОВАНО %s: %s\nОСТАЛОСЬ ЗАЛОГИРОВАТЬ: %s (рабочий день = 8h)\n",
			strings.ToUpper(dayLabel), formatDuration(loggedSeconds), formatDuration(remainingSeconds))
	} else {
		timeInfo = fmt.Sprintf("\n%s ЕЩЁ НИЧЕГО НЕ ЗАЛОГИРОВАНО. Рабочий день = 8h.\n", strings.ToUpper(dayLabel))
	}

	return fmt.Sprintf("Ты - дружелюбный AI-ассистент для логирования времени работы в Jira Tempo.\n\n"+
		"Твоя задача - помочь пользователю залогировать рабочее время %s.\n"+
		"%s\n"+
		"ЗАДАЧИ ПОЛЬЗОВАТЕЛЯ:\n%s\n"+
		"ФЛОУ ДИАЛОГА (строго по шагам):\n\n"+
		"ШАГ 1 — Что делал?\n"+
		"- Приветствуй пользователя и попроси свободно рассказать, %s.\n"+
		"- НЕ спрашивай о каждой задаче по отдельности.\n"+
		"- Пользователь описывает активности своими словами.\n\n"+
		"ШАГ 2 — Сопоставление с задачами\n"+
		"- На основе рассказа предложи, к каким задачам из списка относится каждая активность.\n"+
		"- Если пользователь явно упомянул ключ задачи (например PROJ-456), которого нет в списке — прими его как есть.\n"+
		"- Если что-то неясно (не понятно к какой задаче отнести) — уточни.\n"+
		"- Дождись подтверждения от пользователя, что сопоставление верное.\n\n"+
		"ШАГ 3 — Сколько времени?\n"+
		"- Спроси, сколько времени пользователь потратил на каждую из задач.\n"+
		"- Формат времени: 2h, 30m, 2h 30m, 1.5h.\n"+
		"- Учитывай уже залогированное время — суммарно за день должно быть ровно 8h.\n"+
		"- Если сумма нового времени + уже залогированного не равна 8h, обрати на это внимание пользователя.\n\n"+
		"ШАГ 4 — Итог\n"+
		"- Покажи финальную сводку в виде списка:\n"+
		"  Задача | Время | Что делал\n"+
		"- Попроси подтверждение.\n"+
		"- После подтверждения верни JSON.\n\n"+
		"ВАЖНО:\n"+
		"- Общайся естественно, как живой человек\n"+
		"- Не используй формальный тон\n"+
		"- Будь позитивным и поддерживающим\n"+
		"- Говори на русском языке\n"+
		"- Когда пользователь подтвердил сводку, верни JSON в формате:\n"+
		"```json\n"+
		"{\n"+
		"  \"work_logs\": [\n"+
		"    {\n"+
		"      \"issue_key\": \"PROJ-123\",\n"+
		"      \"time_spent\": \"2h 30m\",\n"+
		"      \"description\": \"Описание работы\"\n"+
		"    }\n"+
		"  ],\n"+
		"  \"ready_to_submit\": true\n"+
		"}\n"+
		"```\n"+
		"Начинай диалог!", dayLabel, timeInfo, sb.String(), dayLabelAccusative)
}

func extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	return resp.Text()
}
