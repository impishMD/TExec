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
	Success  string `json:"success"`
	Error    string `json:"error"`
	Timezone string `json:"timezone"`
	Locale   string `json:"locale"`
}

func Defaults(locale string) Settings {
	s := Settings{Locale: "en", Timezone: "UTC"}
	if locale == "ru" {
		s.Locale = "ru"
		s.Success = "✅ {{ task.status }}\n{{ project.name }} · {{ template.name }}\nЗапуск #{{ task.id }} · {{ task.started_by }}\n{{ date }} {{ time }} ({{ timezone }})\n{{ task.url }}"
		s.Error = "❌ {{ task.status }}\n{{ project.name }} · {{ template.name }}\nЗапуск #{{ task.id }} · {{ task.started_by }}\n{{ date }} {{ time }} ({{ timezone }})\n{{ task.error }}\n{{ task.url }}"
	} else {
		s.Success = "✅ {{ task.status }}\n{{ project.name }} · {{ template.name }}\nRun #{{ task.id }} · {{ task.started_by }}\n{{ date }} {{ time }} ({{ timezone }})\n{{ task.url }}"
		s.Error = "❌ {{ task.status }}\n{{ project.name }} · {{ template.name }}\nRun #{{ task.id }} · {{ task.started_by }}\n{{ date }} {{ time }} ({{ timezone }})\n{{ task.error }}\n{{ task.url }}"
	}
	return s
}

func (s Settings) Validate() error {
	if s.Locale != "en" && s.Locale != "ru" {
		return common_errors.NewValidationError("unsupported notification language")
	}
	for _, text := range []string{s.Success, s.Error} {
		if err := notificationmessage.Validate(text, s.Timezone, Variables); err != nil {
			return common_errors.NewValidationError(err.Error())
		}
	}
	return nil
}

// Nil retains the legacy notification format for existing projects.
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
	text, label := s.Success, "Success"
	switch status {
	case "success":
		if s.Locale == "ru" {
			label = "Успешно"
		}
	case "error":
		text, label = s.Error, "Error"
		if s.Locale == "ru" {
			label = "Ошибка"
		}
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
