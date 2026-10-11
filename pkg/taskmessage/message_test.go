package taskmessage

import (
	"encoding/json"
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
	s.SuccessLocale = "en"
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
	s.SuccessLocale = "xx"
	require.Error(t, s.Validate())
	parsed, err := Parse(nil)
	require.NoError(t, err)
	require.Nil(t, parsed)
	_, err = Parse(new(`{"success":""}`))
	require.Error(t, err)
}

func TestDefaultsAndLocaleSnapshots(t *testing.T) {
	require.Len(t, AllDefaults(), 15)
	for locale, settings := range AllDefaults() {
		require.NoError(t, settings.Validate(), locale)
		for _, status := range []string{"success", "error"} {
			text, err := settings.Render(status, time.Now(), SampleValues(locale, "Project", "admin", status))
			require.NoError(t, err, locale)
			require.NotContains(t, text, "{{", locale)
		}
		raw, err := json.Marshal(settings)
		require.NoError(t, err)
		restored, err := Parse(new(string(raw)))
		require.NoError(t, err)
		require.Equal(t, settings, *restored)
	}
	require.Equal(t, "zh_cn", NormalizeLocale("zh-CN"))
	require.Equal(t, "ja", NormalizeLocale("ja-JP"))
	require.Equal(t, "pt_br", NormalizeLocale("pt-BR"))
	// A reset of success in Japanese must leave the Russian error template intact.
	saved := Defaults("ru")
	japanese := Defaults("ja")
	saved.Success, saved.SuccessLocale = japanese.Success, japanese.SuccessLocale
	success, err := saved.Render("success", time.Now(), nil)
	require.NoError(t, err)
	require.Contains(t, success, "成功")
	failure, err := saved.Render("error", time.Now(), nil)
	require.NoError(t, err)
	require.Contains(t, failure, "Ошибка")
	legacy := `{"success":"Мой {{task.status}}","error":"Ошибка","timezone":"UTC","locale":"ru"}`
	parsed, err := Parse(&legacy)
	require.NoError(t, err)
	require.Equal(t, "ru", parsed.SuccessLocale)
	require.Equal(t, "ru", parsed.ErrorLocale)
	require.Equal(t, "Мой {{task.status}}", parsed.Success)
}
