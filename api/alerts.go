package api

import (
	"net/http"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/notificationmessage"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/pkg/tz"
	"github.com/impishMD/TExec/services/alerting"
	"github.com/impishMD/TExec/services/audit"
)

// Project settings contain credentials, so reads also require permission to
// update the project. Tokens are write-only even for administrators.
func alertSettings(w http.ResponseWriter, r *http.Request) {
	user := helpers.GetFromContext(r, "user").(*db.User)
	projectID := 0
	if value, ok := helpers.GetOkFromContext(r, "project"); ok {
		projectID = value.(db.Project).ID
		permissions := helpers.GetFromContext(r, "permissions").(db.ProjectUserPermission)
		if !user.Admin && permissions&db.CanUpdateProject == 0 {
			helpers.RecordDenied(r, "update_project", projectID)
			w.WriteHeader(http.StatusForbidden)
			return
		}
	} else if !user.Admin {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	service := alerting.Service{Store: helpers.Store(r)}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		settings, err := service.GetTelegram(projectID)
		if err != nil {
			helpers.WriteError(w, err)
			return
		}
		helpers.WriteJSON(w, http.StatusOK, settings)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var update struct {
		alerting.TelegramUpdate
		Locale string `json:"locale"`
		Status string `json:"status"`
	}
	if !helpers.Bind(w, r, &update) {
		return
	}
	if r.Method == http.MethodPost {
		var err error
		if projectID != 0 {
			if !update.Enabled {
				helpers.WriteErrorStatus(w, "Telegram notifications are disabled for this project", http.StatusBadRequest)
				return
			}
			settings := update.Templates
			if settings == nil {
				saved, loadErr := service.GetTelegram(projectID)
				if loadErr != nil {
					helpers.WriteError(w, loadErr)
					return
				}
				settings = saved.Templates
			}
			if settings == nil {
				defaults := taskmessage.Defaults(update.Locale)
				settings = &defaults
			}
			if update.Status == "" {
				update.Status = "success"
			}
			project := helpers.GetFromContext(r, "project").(db.Project)
			var text string
			text, err = settings.Render(update.Status, tz.Now(),
				taskmessage.SampleValues(update.Locale, project.Name, user.Username, update.Status))
			if err == nil {
				text = notificationmessage.Truncate(taskmessage.TestPrefix(update.Locale)+text, notificationmessage.MaxMessageRunes)
				err = service.TestTelegramMessage(r.Context(), projectID, update.TelegramUpdate, text, nil)
			}
		} else {
			err = service.TestTelegram(r.Context(), projectID, update.TelegramUpdate, update.Locale, nil)
		}
		if err != nil {
			// Never return Telegram URLs or response bodies: they may contain credentials.
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		helpers.WriteJSON(w, http.StatusOK, map[string]bool{"sent": true})
		return
	}
	settings, err := service.UpdateTelegram(projectID, update.TelegramUpdate)
	if err != nil {
		helpers.WriteError(w, err)
		return
	}
	helpers.Audit(r).Record(r.Context(), audit.Event{
		Kind:      audit.SystemSettingsUpdate,
		ProjectID: projectID,
		Metadata:  audit.SettingsMetadata{Keys: []string{"alerts.telegram"}},
	})
	helpers.WriteJSON(w, http.StatusOK, settings)
}
