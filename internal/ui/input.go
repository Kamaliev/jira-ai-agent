package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/pterm/pterm"
)

type inputModel struct {
	textInput textinput.Model
	submitted bool
	cancelled bool
}

func newInputModel(prompt string) inputModel {
	ti := textinput.New()
	ti.Prompt = pterm.Bold.Sprint(pterm.Cyan(prompt))
	ti.Focus()
	ti.SetSuggestions(CommandNames())
	ti.ShowSuggestions = true
	return inputModel{textInput: ti}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.submitted = true
			return m, tea.Quit
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m inputModel) View() string {
	return m.textInput.View()
}

func ReadInput(prompt string) string {
	m := newInputModel(prompt)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}
	result := finalModel.(inputModel)
	if result.cancelled {
		return ""
	}
	return strings.TrimSpace(result.textInput.Value())
}

func ConfirmYesNo(question string) bool {
	s := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%s [Y/n]: ", pterm.Bold.Sprint(question))
		if !s.Scan() {
			return false
		}
		answer := strings.ToLower(strings.TrimSpace(s.Text()))
		switch answer {
		case "", "y", "yes", "д", "да":
			return true
		case "n", "no", "н", "нет":
			return false
		}
	}
}

func IsExitCommand(input string) bool {
	lower := strings.ToLower(strings.TrimSpace(input))
	switch lower {
	case "выход", "exit", "quit", "стоп":
		return true
	}
	return false
}

func ReadDateRange() (startDate, endDate string, err error) {
	validateDate := func(s string) error {
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return fmt.Errorf("неверный формат даты, используйте ГГГГ-ММ-ДД")
		}
		return nil
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Дата начала (ГГГГ-ММ-ДД)").
				Value(&startDate).
				Validate(validateDate),
			huh.NewInput().
				Title("Дата конца (ГГГГ-ММ-ДД)").
				Value(&endDate).
				Validate(validateDate),
		),
	)

	if err := form.Run(); err != nil {
		return "", "", fmt.Errorf("ввод дат: %w", err)
	}

	return startDate, endDate, nil
}
