package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// PrintTypewriter prints text with a typewriter effect, stripping JSON and markdown.
func PrintTypewriter(text string) {
	text = stripJSON(text)
	text = stripMarkdown(text)
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	fmt.Println()
	pterm.Print(pterm.FgMagenta.Sprint("AI: "))
	for _, ch := range text {
		fmt.Print(string(ch))
		time.Sleep(10 * time.Millisecond)
	}
	fmt.Println()
	fmt.Println()
}

var (
	reBold       = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reItalic     = regexp.MustCompile(`\*(.+?)\*`)
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reHeading    = regexp.MustCompile(`(?m)^#{1,3}\s+`)
)

func stripMarkdown(text string) string {
	text = reBold.ReplaceAllString(text, "$1")
	text = reItalic.ReplaceAllString(text, "$1")
	text = reInlineCode.ReplaceAllString(text, "$1")
	text = reHeading.ReplaceAllString(text, "")
	return text
}

func stripJSON(text string) string {
	if idx := strings.Index(text, "```json"); idx >= 0 {
		return strings.TrimSpace(text[:idx])
	}
	if strings.Contains(text, "```") && strings.Contains(text, "{") {
		return strings.TrimSpace(strings.SplitN(text, "```", 2)[0])
	}
	if strings.Contains(text, "{") && strings.Contains(text, `"work_logs"`) {
		jsonStart := strings.Index(text, "{")
		if jsonStart > 0 {
			return strings.TrimSpace(text[:jsonStart])
		}
	}
	return text
}
