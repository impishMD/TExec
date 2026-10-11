package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/services/server"
	"github.com/stretchr/testify/require"
)

func TestWorkflowNotificationAPIPermissionsAndPreview(t *testing.T) {
	f := newCoreFixture(t)
	svc := server.NewWorkflowService(f.store, f.store, approvalOnlyQueue{}, nil)
	f.router = Route(f.store, nil, f.store, f.store, nil, nil, nil, nil, nil, nil, nil, nil, nil, server.NewRunnerService(f.store), svc, nil)
	base := fmt.Sprintf("/project/%d/workflows", f.project.ID)
	body := `{"notification":{"channel":"telegram","message":"{{ project.name }} / {{ run.id }} / {{ source.error }}","timezone":"Europe/Moscow","locale":"ru"}}`
	for _, endpoint := range []string{"preview", "test"} {
		f.request(t, "guest", "POST", base+"/notifications/"+endpoint, body, 403)
		f.request(t, "outsider", "POST", base+"/notifications/"+endpoint, body, 404)
	}
	response := f.request(t, "manager", "POST", base+"/notifications/preview", body, 200)
	require.Contains(t, response.Body.String(), "42")
	require.Contains(t, response.Body.String(), "Превышено время ожидания")
	require.NotContains(t, response.Body.String(), "secret")
	f.request(t, "manager", "POST", base+"/notifications/preview", `{"notification":{"channel":"telegram","message":"{{ secrets.password }}"}}`, 400)
	response = f.request(t, "manager", "POST", base+"/notifications/test", body, 502)
	require.Contains(t, response.Body.String(), "disabled")
	// Creating and reading a run must expose delivery status, never the rendered payload.
	config := &db.WorkflowNotificationConfig{Channel: "telegram", Message: "{{ run.id }}", Timezone: "UTC"}
	graph, err := f.store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: f.project.ID, Name: "notify", Nodes: []db.WorkflowNode{{ID: 1, Kind: db.WorkflowNodeNotificationKind, Notification: config}}})
	require.NoError(t, err)
	path := fmt.Sprintf("%s/%d/runs", base, graph.ID)
	response = f.request(t, "manager", "POST", path, "", 201)
	var run db.WorkflowRun
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &run))
	_, err = f.store.Sql().Exec("update project__workflow_notification set message=? where workflow_run_id=?", "private-rendered-message", run.ID)
	require.NoError(t, err)
	response = f.request(t, "guest", "GET", fmt.Sprintf("%s/%d", path, run.ID), "", 200)
	require.Contains(t, response.Body.String(), `"status":"queued"`)
	require.NotContains(t, response.Body.String(), "private-rendered-message")
	f.request(t, "admin", "GET", fmt.Sprintf("/project/%d/workflows/%d/runs/%d", f.other.ID, graph.ID, run.ID), "", 404)
}
