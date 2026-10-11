package tasks

import (
	"bytes"
	"encoding/json"
	htmltemplate "html/template"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/db/sql"
	"github.com/impishMD/TExec/pkg/task_logger"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/services/alerting"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/require"
)

type notificationOutputStore struct{ db.WorkflowManager }

func (notificationOutputStore) GetWorkflowTaskOutputTail(projectID, taskID int) ([]string, error) {
	return []string{"ordinary output", "ERROR: connection failed token=private"}, nil
}

type notificationTransport func(*http.Request) (*http.Response, error)

func (f notificationTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProjectMessageDeliveredAndSuppressed(t *testing.T) {
	oldConfig, oldTransport := util.Config, http.DefaultTransport
	store := sql.InitConfigCreateTestStore()
	t.Cleanup(func() { store.Close(); util.Config = oldConfig; http.DefaultTransport = oldTransport })
	project, err := store.CreateProject(db.Project{Name: "Project"})
	require.NoError(t, err)
	settings := taskmessage.Defaults("ru")
	settings.Success = "{{task.status}} {{template.name}} #{{task.id}} {{task.started_by}}"
	settings.Error = "{{task.status}} {{task.error}}"
	runner := &TaskRunner{
		Task:     db.Task{ID: 42, ProjectID: project.ID, Status: task_logger.TaskSuccessStatus, ProjectTokenName: "CI"},
		Template: db.Template{ProjectID: project.ID, Name: "<tag>\n\"quoted\"", Type: db.TemplateTask},
		pool:     &TaskPool{store: store, logger: make(chan logRecord, 20), workflowRepo: notificationOutputStore{}},
	}
	require.Empty(t, runner.projectAlertMessage("telegram"), "existing projects keep legacy formatting")
	raw, marshalErr := json.Marshal(settings)
	require.NoError(t, marshalErr)
	channel, channelErr := store.GetAlertChannel(project.ID, "telegram")
	require.NoError(t, channelErr)
	channel.MessageTemplates = new(string(raw))
	require.NoError(t, store.SetAlertChannel(channel))
	require.Empty(t, runner.projectAlertMessage("slack"), "Telegram text must not leak to another channel")
	text := runner.projectAlertMessage("telegram")
	require.Equal(t, "Успешно <tag>\n\"quoted\" #42 CI", text)
	service := alerting.Service{Store: store}
	_, err = service.UpdateTelegram(project.ID, alerting.TelegramUpdate{Enabled: true, ChatID: "@preview", Token: new("123:test-token")})
	require.NoError(t, err)
	calls := 0
	http.DefaultTransport = notificationTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, text, payload["text"])
		require.NotContains(t, payload, "parse_mode")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	runner.sendTelegramAlert(text)
	require.Equal(t, 1, calls)
	runner.Template.SuppressSuccessAlerts = true
	require.Empty(t, runner.projectAlertMessage("telegram"))
	runner.sendTelegramAlert(text)
	require.Equal(t, 1, calls)
	runner.Task.Status = task_logger.TaskFailStatus
	text = runner.projectAlertMessage("telegram")
	require.Equal(t, "Ошибка ERROR: connection failed token=[redacted]", text)
	runner.sendTelegramAlert(text)
	require.Equal(t, 2, calls)
	runner.Template.SuppressErrorAlerts = true
	runner.sendTelegramAlert(text)
	require.Equal(t, 2, calls)
	runner.Task.Status = task_logger.TaskWaitingConfirmation
	require.Empty(t, runner.projectAlertMessage("telegram"), "success/error templates do not replace approval notifications")
	// Rendering reads saved settings anew, also for already-running tasks.
	settings.Success = "updated"
	raw, err = json.Marshal(settings)
	require.NoError(t, err)
	channel, err = store.GetAlertChannel(project.ID, "telegram")
	require.NoError(t, err)
	channel.MessageTemplates = new(string(raw))
	require.NoError(t, store.SetAlertChannel(channel))
	runner.Template.SuppressSuccessAlerts = false
	runner.Task.Status = task_logger.TaskSuccessStatus
	require.Equal(t, "updated", runner.projectAlertMessage("telegram"))
}

func TestCommonMessageEscapingAcrossTransports(t *testing.T) {
	message := "<script>\"hello\"</script>\nnext \\ line"
	for _, name := range []string{"slack", "rocketchat", "microsoft-teams", "dingtalk", "gotify"} {
		t.Run(name, func(t *testing.T) {
			tpl, err := parseAlertTemplate(name + ".tmpl")
			require.NoError(t, err)
			var body bytes.Buffer
			require.NoError(t, tpl.Execute(&body, Alert{Name: "task", Message: message}))
			var payload any
			require.NoError(t, json.Unmarshal(body.Bytes(), &payload), body.String())
			// Decode every JSON string recursively and ensure the message survives exactly.
			var contains func(any) bool
			contains = func(v any) bool {
				switch value := v.(type) {
				case string:
					return value == message
				case []any:
					for _, item := range value {
						if contains(item) {
							return true
						}
					}
				case map[string]any:
					for _, item := range value {
						if contains(item) {
							return true
						}
					}
				}
				return false
			}
			require.True(t, contains(payload))
			body.Reset()
			require.NoError(t, tpl.Execute(&body, Alert{Name: "task"}))
			require.True(t, json.Valid(body.Bytes()), "legacy payload stays valid")
		})
	}
	tpl, err := htmltemplate.ParseFS(templates, "templates/email.tmpl")
	require.NoError(t, err)
	var body bytes.Buffer
	require.NoError(t, tpl.Execute(&body, Alert{Message: message}))
	require.NotContains(t, body.String(), "<script>")
	require.Contains(t, body.String(), "&lt;script&gt;")
}

func TestAlertLogBarrierDoesNotFinishAnsibleSummary(t *testing.T) {
	// The nil-task barrier must acknowledge once pending output has been persisted.
	pool := &TaskPool{logger: make(chan logRecord, 2), logsReady: make(chan struct{}), logsDone: make(chan struct{}), stop: make(chan struct{})}
	go pool.handleLogs()
	<-pool.logsReady
	pool.syncLogs(nil, time.Now())
	close(pool.stop)
	<-pool.logsDone
}
