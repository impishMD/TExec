package workflowmessage

import (
	"github.com/impishMD/TExec/pkg/notificationmessage"
	"time"
)

const MaxTemplateRunes = notificationmessage.MaxTemplateRunes
const MaxMessageRunes = notificationmessage.MaxMessageRunes

var Variables = []string{
	"project.name", "workflow.name", "run.id", "run.started_by", "run.url",
	"date", "time", "timezone", "source.name", "source.status", "source.error",
	"approval.deadline",
}

func Validate(message, timezone string) error {
	return notificationmessage.Validate(message, timezone, Variables)
}
func Render(message, timezone string, at time.Time, values map[string]string) (string, error) {
	return notificationmessage.Render(message, timezone, at, values, Variables)
}
func Truncate(value string, limit int) string { return notificationmessage.Truncate(value, limit) }
func ErrorSummary(outputs []string) string    { return notificationmessage.ErrorSummary(outputs) }
