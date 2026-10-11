package api

import (
	"bytes"
	"encoding/json"
	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/db/sql"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/services/alerting"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChannelTemplatesAccessPersistenceAndPreview(t *testing.T) {
	oldConfig, oldTransport := util.Config, http.DefaultTransport
	store := sql.InitConfigCreateTestStore()
	t.Cleanup(func() { store.Close(); util.Config = oldConfig; http.DefaultTransport = oldTransport })
	project, err := store.CreateProject(db.Project{Name: "sample project"})
	require.NoError(t, err)
	settings := taskmessage.Defaults("ru")
	settings.Success = "<b>{{ project.name }}</b> {{task.status}} #{{task.id}}"
	body := map[string]any{"templates": settings, "status": "success", "locale": "ja",
		"enabled": true, "chat_id": "@unsaved", "token": "123:unsaved-token"}
	request := func(admin bool, permission db.ProjectUserPermission, method, path string) *httptest.ResponseRecorder {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		r := httptest.NewRequest(method, "/api/project/1/alerts/telegram"+path, bytes.NewReader(data))
		r = helpers.SetContextValue(r, "store", store)
		r = helpers.SetContextValue(r, "user", &db.User{Admin: admin, Username: "tester"})
		r = helpers.SetContextValue(r, "project", project)
		r = helpers.SetContextValue(r, "permissions", permission)
		w := httptest.NewRecorder()
		if path == "/templates/preview" {
			projectAlertTemplatePreview(w, r)
		} else {
			alertSettings(w, r)
		}
		return w
	}
	for _, route := range [][2]string{{"GET", ""}, {"HEAD", ""}, {"PUT", ""}, {"POST", "/templates/preview"}, {"POST", "/test"}} {
		for _, permission := range []db.ProjectUserPermission{0, db.CanManageProjectResources, db.CanRunProjectTasks} {
			require.Equal(t, 403, request(false, permission, route[0], route[1]).Code)
		}
	}
	w := request(true, 0, "GET", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var initial alerting.TelegramSettings
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &initial))
	require.Nil(t, initial.Templates)
	require.Len(t, initial.TemplateDefaults, 15)
	require.Equal(t, taskmessage.Defaults("zh_cn"), initial.TemplateDefaults["zh_cn"])
	w = request(false, db.CanUpdateProject, "POST", "/templates/preview")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "Успешно #42", "viewer language must not alter saved Russian status")
	calls := 0
	http.DefaultTransport = alertTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Contains(t, r.URL.Path, "123:unsaved-token")
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "@unsaved", payload["chat_id"])
		require.Equal(t, taskmessage.TestPrefix("ja")+"<b>sample project</b> Успешно #42", payload["text"])
		require.NotContains(t, payload, "parse_mode")
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))}, nil
	})
	body["enabled"] = false
	require.Equal(t, 400, request(true, 0, "POST", "/test").Code)
	require.Zero(t, calls)
	body["enabled"] = true
	w = request(false, db.CanUpdateProject, "POST", "/test")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.Equal(t, 1, calls)
	current, err := store.GetAlertChannel(project.ID, "telegram")
	require.NoError(t, err)
	require.Nil(t, current.MessageTemplates)
	require.Empty(t, current.Secret, "testing must not persist credentials")
	w = request(false, db.CanUpdateProject, "PUT", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "unsaved-token")
	saved, err := (alerting.Service{Store: store}).GetTelegram(project.ID)
	require.NoError(t, err)
	require.Equal(t, settings, *saved.Templates)
	require.True(t, saved.Enabled)
	require.True(t, saved.HasToken)
	require.Equal(t, "@unsaved", saved.ChatID)
	settings.Success = "{{ secret }}"
	body["templates"], body["chat_id"] = settings, "@invalid"
	require.Equal(t, 400, request(true, 0, "PUT", "").Code)
	require.Equal(t, 400, request(true, 0, "POST", "/test").Code)
	require.Equal(t, 1, calls)
	unchanged, err := (alerting.Service{Store: store}).GetTelegram(project.ID)
	require.NoError(t, err)
	require.Equal(t, saved, unchanged, "invalid templates must not partially update the channel")
}
