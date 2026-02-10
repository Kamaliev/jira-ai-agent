package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	JiraURL      string `json:"jira_url"`
	JiraEmail    string `json:"jira_email"`
	JiraAPIToken string `json:"jira_api_token"`
	GeminiAPIKey string `json:"gemini_api_key"`
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

func mask(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

func prompt(scanner *bufio.Scanner, label, current string, secret bool) string {
	if current != "" {
		displayed := current
		if secret {
			displayed = mask(current)
		}
		fmt.Printf("%s [%s]: ", label, displayed)
	} else {
		fmt.Printf("%s: ", label)
	}
	scanner.Scan()
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		return current
	}
	return val
}

func RunSetup() (*Config, error) {
	var existing Config
	if cfg, err := LoadFromFile(); err == nil {
		existing = *cfg
	}

	fmt.Println("=== Secretary Configuration ===")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	cfg := &Config{
		JiraURL:      prompt(scanner, "Jira URL", existing.JiraURL, false),
		JiraEmail:    prompt(scanner, "Jira Email", existing.JiraEmail, false),
		JiraAPIToken: prompt(scanner, "Jira API Token", existing.JiraAPIToken, true),
		GeminiAPIKey: prompt(scanner, "Gemini API Key", existing.GeminiAPIKey, true),
	}

	cfg.JiraURL = strings.TrimRight(cfg.JiraURL, "/")

	if err := Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("Config saved to %s\n", configPath())
	return cfg, nil
}
