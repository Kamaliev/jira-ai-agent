package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

const DefaultGeminiModel = "gemini-3-flash-preview"

type Config struct {
	JiraURL      string `json:"jira_url"`
	JiraEmail    string `json:"jira_email"`
	JiraAPIToken string `json:"jira_api_token"`
	GeminiAPIKey string `json:"gemini_api_key"`
	GeminiModel  string `json:"gemini_model"`
}

func GeminiModelOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Gemini 3 Flash", "gemini-3-flash-preview"),
		huh.NewOption("Gemini 2.5 Flash", "gemini-2.5-flash"),
		huh.NewOption("Gemini 2.5 Flash", "gemini-2.5-flash-lite"),
	}
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".secretary")
}

func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

func Exists() bool {
	_, err := os.Stat(configPath())
	return err == nil
}

func LoadFromFile() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}
	cfg.JiraURL = strings.TrimRight(cfg.JiraURL, "/")
	if cfg.GeminiModel == "" {
		cfg.GeminiModel = DefaultGeminiModel
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func RunSetup() (*Config, error) {
	var existing Config
	if cfg, err := LoadFromFile(); err == nil {
		existing = *cfg
	}
	if existing.GeminiModel == "" {
		existing.GeminiModel = DefaultGeminiModel
	}

	cfg := existing

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Jira URL").
				Placeholder("https://your-org.atlassian.net").
				Value(&cfg.JiraURL).
				Validate(func(s string) error {
					if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
						return fmt.Errorf("URL must start with http:// or https://")
					}
					return nil
				}),
			huh.NewInput().
				Title("Jira Email").
				Placeholder("you@company.com").
				Value(&cfg.JiraEmail).
				Validate(func(s string) error {
					if !strings.Contains(s, "@") {
						return fmt.Errorf("must be a valid email address")
					}
					return nil
				}),
		).Title("Jira Connection"),

		huh.NewGroup(
			huh.NewInput().
				Title("Jira API Token").
				EchoMode(huh.EchoModePassword).
				Value(&cfg.JiraAPIToken),
			huh.NewInput().
				Title("Gemini API Key").
				EchoMode(huh.EchoModePassword).
				Value(&cfg.GeminiAPIKey),
		).Title("API Tokens"),

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Gemini Model").
				Options(GeminiModelOptions()...).
				Value(&cfg.GeminiModel),
		).Title("AI Model"),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	cfg.JiraURL = strings.TrimRight(cfg.JiraURL, "/")

	if err := Save(&cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\nConfig saved to %s\n", configPath())
	return &cfg, nil
}
