package workflowmessage

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestSubstitutionsArePlainTextAndUseEventTime(t *testing.T) {
	at := time.Date(2026, 10, 10, 22, 30, 0, 0, time.UTC)
	text, err := Render("{{ date }} {{time}} {{ timezone }}\n{{ project.name }} {{ source.error }}", "Europe/Moscow", at, map[string]string{"project.name": "<b>{{ run.id }}</b>"})
	require.NoError(t, err)
	require.Equal(t, "11.10.2026 01:30:00 Europe/Moscow\n<b>{{ run.id }}</b> —", text)
	for _, message := range []string{"", "{{ secrets }}", "{{ run.password }}", "{{ run.id", "{{ if run.id }}", "{{ .Run.ID }}", strings.Repeat("я", MaxTemplateRunes+1)} {
		require.Error(t, Validate(message, "UTC"), message)
	}
	require.Error(t, Validate("test", "not/a/timezone"))
	text, err = Render("{{ source.error }}", "", at, map[string]string{"source.error": strings.Repeat("я", 5000)})
	require.NoError(t, err)
	require.True(t, utf8.ValidString(text))
	require.Equal(t, MaxMessageRunes, utf8.RuneCountInString(text))
}

func TestErrorSummaryDoesNotSendArbitraryOutputAndMasksCredentials(t *testing.T) {
	text := ErrorSummary([]string{
		"ordinary log with unrecognized private data",
		"\x1b[31m[ERROR]: password=secret token=abc Authorization: Bearer abc.def.ghi\x1b[0m",
		"Error: failed https://user:secret@example.test/path",
		"[ERROR]: api_key=xyz",
	})
	for _, hidden := range []string{"ordinary", "=secret", "abc", "ghi", "user:secret", "xyz", "\x1b"} {
		require.NotContains(t, text, hidden)
	}
	require.Contains(t, text, "[redacted]")
	require.Empty(t, ErrorSummary([]string{"a final line with no error marker"}))
}
