# sj — Jira Tempo AI Agent

CLI-инструмент для автоматического логирования рабочего времени в Jira. AI-ассистент на базе Google Gemini проводит короткое интервью о проделанной за день работе и сам записывает ворклоги в Jira.

## Как это работает

1. `sj` подключается к Jira и загружает ваши задачи в статусе **In Progress**
2. AI-ассистент задает вопросы: над чем работали, сколько времени потратили
3. По итогам диалога формируется сводка с ворклогами
4. После подтверждения данные отправляются в Jira

## Требования

- Аккаунт Jira Cloud с [API-токеном](https://id.atlassian.com/manage-profile/security/api-tokens)
- [Google Gemini API Key](https://aistudio.google.com/app/apikey)

## Установка

### Быстрая установка (Linux / macOS)

```bash
curl -sL https://raw.githubusercontent.com/Kamaliev/jira-ai-agent/main/install.sh | sh
```

Конкретная версия:

```bash
curl -sL https://raw.githubusercontent.com/Kamaliev/jira-ai-agent/main/install.sh | sh -s v1.0.0
```

Скрипт автоматически определит ОС и архитектуру, скачает нужный бинарник и установит его в `/usr/local/bin/sj`.

### GitHub Releases

Скачайте бинарник для своей платформы со страницы [Releases](https://github.com/Kamaliev/jira-ai-agent/releases):

| Файл | Платформа |
|---|---|
| `sj-linux-amd64` | Linux x86_64 |
| `sj-linux-arm64` | Linux ARM64 |
| `sj-darwin-amd64` | macOS Intel |
| `sj-darwin-arm64` | macOS Apple Silicon |
| `sj-windows-amd64.exe` | Windows x86_64 |

Далее:

```bash
# Linux / macOS
chmod +x sj-*
sudo mv sj-* /usr/local/bin/sj
```

```powershell
# Windows — переместите sj-windows-amd64.exe в папку из PATH и переименуйте в sj.exe
```

### Сборка из исходников

```bash
git clone https://github.com/Kamaliev/jira-ai-agent.git
cd jira-ai-agent
make build
# Бинарник: bin/sj
```

Требуется Go 1.24+.

## Настройка

При первом запуске `sj` запустит интерактивный мастер настройки:

```
$ sj

No configuration found. Let's set it up!

=== Secretary Configuration ===

Jira URL: https://your-company.atlassian.net
Jira Email: your-email@company.com
Jira API Token: ****
Gemini API Key: ****

Config saved to ~/.secretary/config.json
```

Конфигурация сохраняется в `~/.secretary/config.json` с правами `0600`.

### Параметры конфигурации

| Параметр | Описание | Пример |
|---|---|---|
| Jira URL | Адрес вашего Jira Cloud | `https://company.atlassian.net` |
| Jira Email | Email аккаунта Jira | `user@company.com` |
| Jira API Token | [API-токен Jira](https://id.atlassian.com/manage-profile/security/api-tokens) | `ATATT3x...` |
| Gemini API Key | [Ключ Google Gemini](https://aistudio.google.com/app/apikey) | `AIza...` |

### Изменение конфигурации

```bash
sj config
```

Покажет текущие значения (токены замаскированы) и позволит их обновить. Нажмите Enter, чтобы оставить текущее значение.

## Использование

```bash
sj
```

Пример сессии:

```
┌──────────────────────┐
│ Jira Tempo AI Agent  │
└──────────────────────┘
Ваш AI-ассистент на базе Google Gemini

 SUCCESS  Найдено задач: 3

┌───┬────────────┬──────────────────────────────┐
│ # │ Ключ       │ Название                     │
├───┼────────────┼──────────────────────────────┤
│ 1 │ PROJ-123   │ Реализовать авторизацию      │
│ 2 │ PROJ-456   │ Исправить баг в отчётах      │
│ 3 │ PROJ-789   │ Обновить документацию API     │
└───┴────────────┴──────────────────────────────┘

Привет! Расскажи, чем сегодня занимался?

Ты: Работал над авторизацией часа 3, и час фиксил баг в отчётах

...

┌────────────┬───────┬──────────────────────────┐
│ Задача     │ Время │ Описание                 │
├────────────┼───────┼──────────────────────────┤
│ PROJ-123   │ 3.0ч  │ Реализация авторизации   │
│ PROJ-456   │ 1.0ч  │ Исправление бага         │
├────────────┼───────┼──────────────────────────┤
│ ИТОГО      │ 4.0ч  │                          │
└────────────┴───────┴──────────────────────────┘

Отправить эти данные в Jira? (да/нет): да

 SUCCESS  PROJ-123
 SUCCESS  PROJ-456

Спасибо! До встречи!
```

### Команды

| Команда | Описание |
|---|---|
| `sj` | Запуск интервью |
| `sj config` | Настройка/изменение конфигурации |
| `sj version` | Показать версию |

### Выход из диалога

Во время интервью можно ввести: `выход`, `exit`, `quit`, `стоп`.

### Поддерживаемые форматы времени

При общении с ассистентом можно указывать время в любом удобном формате:

- `2h 30m`, `2h30m`
- `1.5h`, `0.5h`
- `2ч 30м`, `1.5ч`
- `30m`, `45м`

## Создание релиза

Релизы создаются автоматически через GitHub Actions при пуше тега:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Workflow соберет бинарники для всех платформ и создаст GitHub Release с артефактами и чексуммами.

## Структура проекта

```
cmd/secretary/main.go    — точка входа
internal/
  config/config.go       — управление конфигурацией (~/.secretary/config.json)
  gemini/assistant.go    — интеграция с Google Gemini AI
  gemini/types.go        — типы данных для ворклогов
  jira/client.go         — клиент Jira REST API v2
  jira/types.go          — типы данных Jira
  session/interview.go   — оркестрация интервью
  timeparse/parse.go     — парсинг строк времени ("2h 30m" -> секунды)
  ui/display.go          — отображение таблиц и сообщений
  ui/input.go            — ввод пользователя
  ui/typewriter.go       — эффект печатной машинки для ответов AI
```

## Лицензия

MIT
