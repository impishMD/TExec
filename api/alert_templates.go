package api

import (
	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/pkg/tz"
	"net/http"
)

// Preview accepts the channel's unsaved form and never sends or persists anything.
func projectAlertTemplatePreview(w http.ResponseWriter, r *http.Request) {
	project := helpers.GetFromContext(r, "project").(db.Project)
	permissions := helpers.GetFromContext(r, "permissions").(db.ProjectUserPermission)
	user := helpers.UserFromContext(r)
	if !user.Admin && permissions&db.CanUpdateProject == 0 {
		helpers.RecordDenied(r, "update_project", project.ID)
		w.WriteHeader(http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var body struct {
		Templates taskmessage.Settings `json:"templates"`
		Status    string               `json:"status"`
		Locale    string               `json:"locale"`
	}
	if !helpers.Bind(w, r, &body) {
		return
	}
	text, err := body.Templates.Render(body.Status, tz.Now(),
		taskmessage.SampleValues(body.Locale, project.Name, user.Username, body.Status))
	if err != nil {
		helpers.WriteErrorStatus(w, err.Error(), http.StatusBadRequest)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"text": text})
}
