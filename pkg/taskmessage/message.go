// Package taskmessage defines channel-independent task result notifications.
package taskmessage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/impishMD/TExec/pkg/common_errors"
	"github.com/impishMD/TExec/pkg/notificationmessage"
)

var Variables = []string{
	"project.name", "template.name", "task.id", "task.status", "task.started_by",
	"task.url", "task.message", "task.version", "task.error", "date", "time", "timezone",
}

type Settings struct {
	Success  string `json:"success" backup:"success"`
	Error    string `json:"error" backup:"error"`
	Timezone string `json:"timezone" backup:"timezone"`
	// Locale snapshots belong to each saved text, not to the viewer. Only an
	// explicit reset changes them; editing unrelated channel settings does not.
	SuccessLocale string `json:"success_locale" backup:"success_locale"`
	ErrorLocale   string `json:"error_locale" backup:"error_locale"`
}

func (s *Settings) UnmarshalJSON(data []byte) error {
	type plain Settings
	var value struct {
		plain
		Locale string `json:"locale" backup:"locale"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*s = Settings(value.plain)
	// Read templates saved by the first local preview without changing their text.
	if s.SuccessLocale == "" {
		s.SuccessLocale = NormalizeLocale(value.Locale)
	}
	if s.ErrorLocale == "" {
		s.ErrorLocale = NormalizeLocale(value.Locale)
	}
	return nil
}

func Defaults(locale string) Settings {
	locale = NormalizeLocale(locale)
	words := translations[locale]
	body := "\n{{ project.name }} · {{ template.name }}\n" + words.Run + " #{{ task.id }} · {{ task.started_by }}\n{{ date }} {{ time }} ({{ timezone }})\n"
	return Settings{
		Success:  "✅ {{ task.status }}" + body + "{{ task.url }}",
		Error:    "❌ {{ task.status }}" + body + "{{ task.error }}\n{{ task.url }}",
		Timezone: "UTC", SuccessLocale: locale, ErrorLocale: locale,
	}
}

func AllDefaults() map[string]Settings {
	result := make(map[string]Settings, len(translations))
	for locale := range translations {
		result[locale] = Defaults(locale)
	}
	return result
}

func (s Settings) Validate() error {
	_, successOK := translations[s.SuccessLocale]
	_, errorOK := translations[s.ErrorLocale]
	if !successOK || !errorOK {
		return common_errors.NewValidationError("unsupported notification language")
	}
	for _, text := range []string{s.Success, s.Error} {
		if err := notificationmessage.Validate(text, s.Timezone, Variables); err != nil {
			return common_errors.NewValidationError(err.Error())
		}
	}
	return nil
}

// Nil retains the legacy notification format for unconfigured channels.
func Parse(raw *string) (*Settings, error) {
	if raw == nil {
		return nil, nil
	}
	var settings Settings
	if err := json.Unmarshal([]byte(*raw), &settings); err != nil {
		return nil, common_errors.NewValidationError("invalid notification templates")
	}
	return &settings, settings.Validate()
}

func (s Settings) Render(status string, at time.Time, values map[string]string) (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	text, label := s.Success, translations[s.SuccessLocale].Success
	switch status {
	case "success":
	case "error":
		text, label = s.Error, translations[s.ErrorLocale].Error
	default:
		return "", fmt.Errorf("unsupported notification status")
	}
	data := make(map[string]string, len(values)+1)
	for key, value := range values {
		data[key] = value
	}
	data["task.status"] = label
	return notificationmessage.Render(text, s.Timezone, at, data, Variables)
}
