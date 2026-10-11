// Package workflowmessage renders a deliberately small, plain-text template language.
// It supports named substitutions only, never expressions or executable templates.
package workflowmessage

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxTemplateRunes = 3000
const MaxMessageRunes = 4000

var Variables = []string{
	"project.name", "workflow.name", "run.id", "run.started_by", "run.url",
	"date", "time", "timezone", "source.name", "source.status", "source.error",
	"approval.deadline",
}

var placeholder = regexp.MustCompile(`\{\{\s*([a-z_]+(?:\.[a-z_]+)?)\s*\}\}`)

func Validate(message, timezone string) error {
	if strings.TrimSpace(message) == "" || utf8.RuneCountInString(message) > MaxTemplateRunes {
		return fmt.Errorf("notification message must contain 1–%d characters", MaxTemplateRunes)
	}
	if timezone == "" {
		timezone = "UTC"
	}
	if len(timezone) > 100 {
		return fmt.Errorf("invalid notification timezone")
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return fmt.Errorf("unknown notification timezone")
	}
	known := map[string]bool{}
	for _, name := range Variables {
		known[name] = true
	}
	for _, match := range placeholder.FindAllStringSubmatch(message, -1) {
		if !known[match[1]] {
			return fmt.Errorf("unknown notification variable: %s", match[1])
		}
	}
	rest := placeholder.ReplaceAllString(message, "")
	if strings.Contains(rest, "{{") || strings.Contains(rest, "}}") {
		return fmt.Errorf("invalid notification variable; use {{ variable.name }}")
	}
	return nil
}

func Render(message, timezone string, at time.Time, values map[string]string) (string, error) {
	if err := Validate(message, timezone); err != nil {
		return "", err
	}
	if timezone == "" {
		timezone = "UTC"
	}
	location, _ := time.LoadLocation(timezone)
	at = at.In(location)
	result := placeholder.ReplaceAllStringFunc(message, func(token string) string {
		name := placeholder.FindStringSubmatch(token)[1]
		switch name {
		case "date":
			return at.Format("02.01.2006")
		case "time":
			return at.Format("15:04:05")
		case "timezone":
			return timezone
		default:
			if value := values[name]; value != "" {
				return value
			}
			return "—"
		}
	})
	return Truncate(result, MaxMessageRunes), nil
}

func Truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}

var ansi = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\))`)
var errorMarker = regexp.MustCompile(`(?i)^\s*[│╷╵]?\s*(?:\[ERROR\]|ERROR(?:[:!\s]|$)|FATAL(?:[:\s]|$)|FAILED(?:!|\s+to\b)|[^\s:]+(?:Error|Exception):)`)
var credential = regexp.MustCompile(`(?i)(["']?(?:[a-z0-9_-]*(?:password|passwd|token|secret|api_key|authorization))["']?\s*[:=]\s*)(?:"[^"\n]*"|'[^'\n]*'|[^\s,;}]+)`)
var bearer = regexp.MustCompile(`(?i)\b(Bearer|Basic)\s+[A-Za-z0-9+/=_\.\-]+`)
var urlCredentials = regexp.MustCompile(`(https?://)[^/\s:@]+:[^/\s@]+@`)

// ErrorSummary includes marked error lines only, not an arbitrary log tail.
// Credential-pattern filtering is best effort; unknown secrets cannot be inferred.
func ErrorSummary(outputs []string) string {
	lines := strings.Split(ansi.ReplaceAllString(strings.Join(outputs, "\n"), ""), "\n")
	var errors []string
	for _, line := range lines {
		if !errorMarker.MatchString(line) {
			continue
		}
		line = bearer.ReplaceAllString(line, "${1} [redacted]")
		line = credential.ReplaceAllString(line, "${1}[redacted]")
		line = urlCredentials.ReplaceAllString(line, "${1}[redacted]@")
		errors = append(errors, Truncate(strings.TrimSpace(line), 500))
		if len(errors) == 3 {
			break
		}
	}
	return Truncate(strings.Join(errors, "\n"), 900)
}
