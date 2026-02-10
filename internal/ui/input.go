package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
)

var scanner = bufio.NewScanner(os.Stdin)

func ReadInput(prompt string) string {
	pterm.Print(pterm.Bold.Sprint(pterm.Cyan(prompt)))
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func ConfirmYesNo(question string) bool {
	for {
		fmt.Printf("%s [Y/n]: ", pterm.Bold.Sprint(question))
		if !scanner.Scan() {
			return false
		}
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
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
