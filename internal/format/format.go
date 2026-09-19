package format

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/a-h/templ"
)

var (
	bold   = regexp.MustCompile(`\*\*([^\n*]+)\*\*`)
	italic = regexp.MustCompile(`_([^\n_]+)_`)
	code   = regexp.MustCompile("`([^\\n`]+)`")
)

func Duration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}

func Minutes(minutes int32) string { return Duration(int64(minutes) * 60) }

func Date(value time.Time) string {
	if value.IsZero() {
		return "—"
	}
	months := []string{"janeiro", "fevereiro", "março", "abril", "maio", "junho", "julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
	return fmt.Sprintf("%d de %s de %d", value.Day(), months[value.Month()-1], value.Year())
}

func ShortDate(value time.Time) string {
	return fmt.Sprintf("%02d/%02d", value.Day(), value.Month())
}

func Relative(value time.Time) string {
	delta := time.Since(value)
	if delta < time.Minute {
		return "agora"
	}
	if delta < time.Hour {
		return fmt.Sprintf("há %d min", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("há %d h", int(delta.Hours()))
	}
	return ShortDate(value)
}

// Markdown renders a deliberately small, escaped Markdown subset.
func Markdown(value string) templ.Component {
	escaped := html.EscapeString(strings.TrimSpace(value))
	escaped = code.ReplaceAllString(escaped, "<code>$1</code>")
	escaped = bold.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = italic.ReplaceAllString(escaped, "<em>$1</em>")
	paragraphs := strings.Split(escaped, "\n\n")
	for index, paragraph := range paragraphs {
		paragraphs[index] = "<p>" + strings.ReplaceAll(paragraph, "\n", "<br>") + "</p>"
	}
	return templ.Raw(strings.Join(paragraphs, ""))
}
