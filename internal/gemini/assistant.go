package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go-secretary/internal/jira"
	"go-secretary/internal/timeparse"

	"google.golang.org/genai"
)

type Assistant struct {
	client *genai.Client
	chat   *genai.Chat
}

func NewAssistant(ctx context.Context, apiKey string) (*Assistant, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}
	return &Assistant{client: client}, nil
}

func (a *Assistant) StartInterview(ctx context.Context, issues []jira.Issue) (string, error) {
	systemPrompt := buildSystemPrompt(issues)

	var err error
	a.chat, err = a.client.Chats.Create(ctx, "models/gemini-3-flash-preview", &genai.GenerateContentConfig{
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
		if !strings.Contains(err.Error(), "429") && !strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			return nil, err
		}
		wait := time.Duration(30*(attempt+1)) * time.Second
		fmt.Printf("\nRate limit hit, retrying in %v...\n", wait)
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

func (a *Assistant) Close() {
	// genai client doesn't require explicit close
}

func buildSystemPrompt(issues []jira.Issue) string {
	var sb strings.Builder
	for _, issue := range issues {
		fmt.Fprintf(&sb, "- %s: %s\n", issue.Key, issue.Summary)
	}

	return fmt.Sprintf(`Ты - дружелюбный AI-ассистент для логирования времени работы в Jira Tempo.

Твоя задача - провести интервью с пользователем о его работе за сегодня.

ДОСТУПНЫЕ ЗАДАЧИ В РАБОТЕ:
%s
ФОРМАТ РАБОТЫ:
1. Приветствуй пользователя тепло и дружелюбно
2. Объясни, что будешь спрашивать о каждой задаче
3. Для каждой задачи спроси:
   - Работал ли он над ней сегодня?
   - Сколько времени потратил? (формат: 2h, 30m, 2h 30m, 1.5h)
4. В конце собери все данные и попроси подтверждение

ВАЖНО:
- Общайся естественно, как живой человек
- Не используй формальный тон
- Задавай уточняющие вопросы при необходимости
- Будь позитивным и поддерживающим
- Говори на русском языке
- Когда собрал всю информацию, верни JSON в формате:
`+"```json\n"+`{
  "work_logs": [
    {
      "issue_key": "PROJ-123",
      "time_spent": "2h 30m",
      "description": "Описание работы"
    }
  ],
  "ready_to_submit": true
}
`+"```\n"+`
Начинай диалог!`, sb.String())
}

func extractText(resp *genai.GenerateContentResponse) string {
	if resp == nil || len(resp.Candidates) == 0 {
		return ""
	}
	return resp.Text()
}
