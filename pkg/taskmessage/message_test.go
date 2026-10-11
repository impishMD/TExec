package taskmessage

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestTaskMessages(t *testing.T) {
	s := Defaults("ru")
	s.Timezone = "Europe/Moscow"
	at := time.Date(2026, 10, 10, 22, 30, 0, 0, time.UTC)
	values := map[string]string{"task.id": "42", "template.name": "<b>{{ task.id }}</b>"}
	text, err := s.Render("success", at, values)
	require.NoError(t, err)
	require.Contains(t, text, "✅ Успешно")
	require.Contains(t, text, "11.10.2026 01:30:00 (Europe/Moscow)")
	require.Contains(t, text, "<b>{{ task.id }}</b>", "values must not be evaluated recursively")
	require.Contains(t, text, "Запуск #42 · —")
	values["task.error"] = "Error: unavailable"
	text, err = s.Render("error", at, values)
	require.NoError(t, err)
	require.Contains(t, text, "❌ Ошибка")
	require.Contains(t, text, "Error: unavailable")
	_, err = s.Render("running", at, values)
	require.Error(t, err)
	s.Locale = "en"
	s.Success = "{{ task.status }} {{ task.message }}"
	values["task.message"] = strings.Repeat("я", 5000)
	text, err = s.Render("success", at, values)
	require.NoError(t, err)
	require.True(t, utf8.ValidString(text))
	require.Equal(t, 4000, utf8.RuneCountInString(text))
	for _, text := range []string{"", "{{run.id}}", "{{task.secret}}", "{{ task.id", "{{ if task.id }}", strings.Repeat("я", 3001)} {
		s.Success = text
		require.Error(t, s.Validate())
	}
	s = Defaults("en")
	s.Timezone = "nowhere/invalid"
	require.Error(t, s.Validate())
	s = Defaults("en")
	s.Locale = "xx"
	require.Error(t, s.Validate())
	parsed, err := Parse(nil)
	require.NoError(t, err)
	require.Nil(t, parsed)
	_, err = Parse(new(`{"success":""}`))
	require.Error(t, err)
}
