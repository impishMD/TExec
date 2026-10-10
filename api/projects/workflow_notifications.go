package projects

import (
	"net/http"
	"strings"
	"time"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/tz"
	"github.com/impishMD/TExec/pkg/workflowmessage"
	"github.com/impishMD/TExec/services/alerting"
)

// Preview/test use synthetic run data only and require project resource management.
// They never read another run, execute a graph, or persist the submitted settings.
func (c *WorkflowController) PreviewNotification(w http.ResponseWriter, r *http.Request) {
	c.notificationPreview(w, r, false)
}
func (c *WorkflowController) TestNotification(w http.ResponseWriter, r *http.Request) {
	c.notificationPreview(w, r, true)
}
func (c *WorkflowController) notificationPreview(w http.ResponseWriter, r *http.Request, send bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var body struct {
		Notification db.WorkflowNotificationConfig `json:"notification"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	if err := body.Notification.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}
	project := helpers.GetFromContext(r, "project").(db.Project)
	user := helpers.UserFromContext(r)
	locale := body.Notification.Locale
	ru := strings.HasPrefix(locale, "ru")
	values := map[string]string{
		"project.name": project.Name, "workflow.name": "Deployment", "run.id": "42",
		"run.started_by": user.Username, "run.url": "https://texec.example.com/project/1/workflows/1/runs/42",
		"source.name": "Infrastructure plan", "source.status": "error", "source.error": "[ERROR]: Connection timed out",
	}
	if ru {
		values["workflow.name"], values["source.name"], values["source.status"] = "Развёртывание", "План инфраструктуры", "Ошибка"
		values["source.error"] = "[ERROR]: Превышено время ожидания соединения"
	}
	location, _ := time.LoadLocation(body.Notification.Timezone)
	if location == nil {
		location = time.UTC
	}
	values["approval.deadline"] = tz.Now().Add(time.Hour).In(location).Format("02.01.2006 15:04:05 MST")
	text, err := workflowmessage.Render(body.Notification.Message, body.Notification.Timezone, tz.Now(), values)
	if err != nil {
		helpers.WriteErrorStatus(w, err.Error(), http.StatusBadRequest)
		return
	}
	if send {
		prefix := "TExec: test notification (sample data)\n\n"
		if ru {
			prefix = "TExec: тестовое оповещение (пример данных)\n\n"
		}
		err = (alerting.Service{Store: helpers.Store(r)}).SendWorkflowNotification(r.Context(), project.ID, workflowmessage.Truncate(prefix+text, workflowmessage.MaxMessageRunes), "", "", nil)
		if err != nil {
			helpers.WriteErrorStatus(w, err.Error(), http.StatusBadGateway)
			return
		}
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{"text": text, "variables": workflowmessage.Variables})
}
