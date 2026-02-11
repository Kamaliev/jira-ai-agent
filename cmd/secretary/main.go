package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"go-secretary/internal/config"
	"go-secretary/internal/gemini"
	"go-secretary/internal/jira"
	"go-secretary/internal/session"

	"github.com/pterm/pterm"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println("sj version", Version)
		return
	}

	if len(os.Args) > 1 && os.Args[1] == "config" {
		if _, err := config.RunSetup(); err != nil {
			pterm.Error.Println(err.Error())
			os.Exit(1)
		}
		return
	}

	cfg, err := config.LoadFromFile()
	if err != nil {
		if !config.Exists() {
			fmt.Println("No configuration found. Let's set it up!")
			fmt.Println()
			cfg, err = config.RunSetup()
			if err != nil {
				pterm.Error.Println(err.Error())
				os.Exit(1)
			}
		} else {
			pterm.Error.Println("Failed to load config: " + err.Error())
			os.Exit(1)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	jiraClient := jira.NewClient(cfg.JiraURL, cfg.JiraEmail, cfg.JiraAPIToken)

	geminiAssistant, err := gemini.NewAssistant(ctx, cfg.GeminiAPIKey)
	if err != nil {
		pterm.Error.Println("Ошибка при инициализации Gemini: " + err.Error())
		os.Exit(1)
	}
	defer geminiAssistant.Close()

	runner := session.NewRunner(jiraClient, geminiAssistant)
	if err := runner.Run(ctx); err != nil {
		if ctx.Err() != nil {
			pterm.Println()
			pterm.Println(pterm.Gray("Прервано пользователем. До встречи!"))
			os.Exit(0)
		}
		pterm.Error.Println(err.Error())
		os.Exit(1)
	}
}
