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
	return value.Format("Jan 2, 2006")
}

func Relative(value time.Time) string {
	delta := time.Since(value)
	if delta < time.Minute {
		return "now"
	}
	if delta < time.Hour {
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	}
	return value.Format("Jan 2")
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

