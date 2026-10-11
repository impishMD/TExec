package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/db/sql"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/services/alerting"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/require"
)

func TestProjectAlertTemplatesAccessPersistenceAndPreview(t *testing.T) {
	oldConfig, oldTransport := util.Config, http.DefaultTransport
	store := sql.InitConfigCreateTestStore()
	t.Cleanup(func() { store.Close(); util.Config = oldConfig; http.DefaultTransport = oldTransport })
	project, err := store.CreateProject(db.Project{Name: "sample project"})
	require.NoError(t, err)
	settings := taskmessage.Defaults("ru")
	settings.Success = "<b>{{ project.name }}</b> {{task.status}} #{{task.id}}"
	request := func(admin bool, permission db.ProjectUserPermission, method, path string, s taskmessage.Settings, status string) *httptest.ResponseRecorder {
		data, err := json.Marshal(map[string]any{"settings": s, "status": status})
		require.NoError(t, err)
		r := httptest.NewRequest(method, "/api/project/1/alerts/templates"+path, bytes.NewReader(data))
		r = helpers.SetContextValue(r, "store", store)
		r = helpers.SetContextValue(r, "user", &db.User{Admin: admin, Username: "tester"})
		r = helpers.SetContextValue(r, "project", project)
		r = helpers.SetContextValue(r, "permissions", permission)
		w := httptest.NewRecorder()
		projectAlertTemplates(w, r)
		return w
	}
	for _, route := range [][2]string{{"GET", ""}, {"HEAD", ""}, {"PUT", ""}, {"POST", "/preview"}, {"POST", "/test"}} {
		for _, permission := range []db.ProjectUserPermission{0, db.CanManageProjectResources, db.CanRunProjectTasks} {
			require.Equal(t, 403, request(false, permission, route[0], route[1], settings, "success").Code)
		}
	}
	w := request(true, 0, "GET", "?locale=ru", settings, "success")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"configured":false`)
	require.Contains(t, w.Body.String(), `"locale":"ru"`)
	for _, route := range []string{"/preview", "/test"} {
		w = request(false, db.CanUpdateProject, "POST", route, settings, "success")
		if route == "/test" {
			require.Equal(t, 400, w.Code, "disabled channel must not report delivery")
		} else {
			require.Equal(t, 200, w.Code, w.Body.String())
			require.Contains(t, w.Body.String(), "Успешно #42")
		}
		current, err := store.GetProject(project.ID)
		require.NoError(t, err)
		require.Nil(t, current.NotificationTemplates)
	}
	service := alerting.Service{Store: store}
	_, err = service.UpdateTelegram(project.ID, alerting.TelegramUpdate{Enabled: true, ChatID: "@preview", Token: new("123:test-token")})
	require.NoError(t, err)
	calls := 0
	http.DefaultTransport = alertTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "@preview", payload["chat_id"])
		require.Equal(t, "TExec: тестовое оповещение (пример данных)\n\n<b>sample project</b> Успешно #42", payload["text"])
		require.NotContains(t, payload, "parse_mode")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	w = request(false, db.CanUpdateProject, "POST", "/test", settings, "success")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, 1, calls)
	current, err := store.GetProject(project.ID)
	require.NoError(t, err)
	require.Nil(t, current.NotificationTemplates)
	w = request(false, db.CanUpdateProject, "PUT", "", settings, "")
	require.Equal(t, 200, w.Code, w.Body.String())
	current, err = store.GetProject(project.ID)
	require.NoError(t, err)
	parsed, err := taskmessage.Parse(current.NotificationTemplates)
	require.NoError(t, err)
	require.Equal(t, settings, *parsed)
	require.Contains(t, request(true, 0, "GET", "", settings, "").Body.String(), `"configured":true`)
	settings.Success = "{{ secret }}"
	require.Equal(t, 400, request(true, 0, "PUT", "", settings, "").Code)
	require.Equal(t, 400, request(true, 0, "POST", "/test", settings, "success").Code)
	require.Equal(t, 1, calls)
}
