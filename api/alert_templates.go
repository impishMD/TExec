package api

import (
	"net/http"
	"strings"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/notificationmessage"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/pkg/tz"
	"github.com/impishMD/TExec/services/alerting"
	"github.com/impishMD/TExec/services/audit"
)

// Reads, previews and test sends have the same authorization as channel settings.
func projectAlertTemplates(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	permissions := helpers.GetFromContext(r, "permissions").(db.ProjectUserPermission)
	user := helpers.UserFromContext(r)
	if !user.Admin && permissions&db.CanUpdateProject == 0 {
		helpers.RecordDenied(r, "update_project", project.ID)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		current, err := helpers.Store(r).GetProject(project.ID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		settings, err := taskmessage.Parse(current.NotificationTemplates)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		configured := settings != nil
		if !configured {
			defaults := taskmessage.Defaults(r.URL.Query().Get("locale"))
			settings = &defaults
		}
		helpers.WriteJSON(w, http.StatusOK, map[string]any{
			"settings": settings, "configured": configured, "variables": taskmessage.Variables,
			"defaults": map[string]taskmessage.Settings{"en": taskmessage.Defaults("en"), "ru": taskmessage.Defaults("ru")},
		})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var body struct {
		Settings taskmessage.Settings `json:"settings"`
		Status   string               `json:"status"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	if err := body.Settings.Validate(); err != nil {
		helpers.WriteError(w, err)
		return
	}
	if r.Method == http.MethodPut {
		if err := helpers.Store(r).SetProjectNotificationTemplates(project.ID, body.Settings); err != nil {
			helpers.WriteError(w, err)
			return
		}
		helpers.Audit(r).Record(r.Context(), audit.Event{
			Kind: audit.SystemSettingsUpdate, ProjectID: project.ID,
			Metadata: audit.SettingsMetadata{Keys: []string{"alerts.templates"}},
		})
		helpers.WriteJSON(w, http.StatusOK, body.Settings)
		return
	}
	// Preview and test deliberately use sample task data and never persist the form.
	values := map[string]string{
		"project.name": project.Name, "template.name": "Deployment", "task.id": "42",
		"task.started_by": user.Username, "task.url": "https://texec.example.com/project/1/templates/1?t=42",
		"task.message": "Application deployment", "task.version": "1.2.3",
	}
	if body.Status == "error" {
		values["task.error"] = "[ERROR]: Connection timed out"
	}
	prefix := "TExec: test notification (sample data)\n\n"
	if body.Settings.Locale == "ru" {
		values["template.name"], values["task.message"] = "Развёртывание", "Развёртывание приложения"
		if body.Status == "error" {
			values["task.error"] = "[ERROR]: Превышено время ожидания соединения"
		}
		prefix = "TExec: тестовое оповещение (пример данных)\n\n"
	}
	text, err := body.Settings.Render(body.Status, tz.Now(), values)
	if err != nil {
		helpers.WriteErrorStatus(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/test") {
		sent, err := (alerting.Service{Store: helpers.Store(r)}).SendTaskNotification(r.Context(), project.ID,
			notificationmessage.Truncate(prefix+text, notificationmessage.MaxMessageRunes), nil)
		if err != nil {
			helpers.WriteErrorStatus(w, err.Error(), http.StatusBadGateway)
			return
		}
		if !sent {
			helpers.WriteErrorStatus(w, "Telegram notifications are disabled for this project", http.StatusBadRequest)
			return
		}
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"text": text})
}
