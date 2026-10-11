package sql

import (
	"testing"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/stretchr/testify/require"
)

func TestWorkflowNotificationStoreIsolationAndDeletion(t *testing.T) {
	store, graph := workflowFixture(t)
	graph.Nodes[1].Kind = db.WorkflowNodeNotificationKind
	graph.Nodes[1].Notification = &db.WorkflowNotificationConfig{Channel: "telegram", Message: "Done", Timezone: "UTC"}
	require.NoError(t, store.UpdateWorkflowTemplate(graph))
	graph, err := store.GetWorkflowTemplate(graph.ProjectID, graph.ID)
	require.NoError(t, err)
	run, err := store.CreateWorkflowRun(db.WorkflowRun{ProjectID: graph.ProjectID, WorkflowTemplateID: graph.ID, RevisionID: graph.RevisionID})
	require.NoError(t, err)
	delivery := db.WorkflowNotification{ProjectID: graph.ProjectID, WorkflowRunID: run.ID, WorkflowNodeID: graph.Nodes[1].ID, Message: "Done"}
	foreign := delivery
	foreign.ProjectID++
	_, err = store.CreateWorkflowNotification(foreign)
	require.ErrorIs(t, err, db.ErrNotFound)
	saved, err := store.CreateWorkflowNotification(delivery)
	require.NoError(t, err)
	_, err = store.CreateWorkflowNotification(delivery)
	require.Error(t, err, "one delivery per run/node")
	items, err := store.GetWorkflowNotifications(graph.ProjectID+1, run.ID)
	require.NoError(t, err)
	require.Empty(t, items)
	foreign = saved
	foreign.ProjectID++
	claimed, err := store.ClaimWorkflowNotification(foreign)
	require.NoError(t, err)
	require.False(t, claimed)
	require.NoError(t, store.CancelWorkflowNotifications(graph.ProjectID+1, run.ID))
	items, err = store.GetWorkflowNotifications(graph.ProjectID, run.ID)
	require.NoError(t, err)
	require.Equal(t, "queued", items[0].Status)
	run.Status = db.WorkflowRunStopped
	require.NoError(t, store.UpdateWorkflowRun(run))
	require.NoError(t, store.DeleteWorkflowTemplate(graph.ProjectID, graph.ID))
	items, err = store.GetWorkflowNotifications(graph.ProjectID, run.ID)
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestMigration_2_20_20PreservesExistingWorkflow(t *testing.T) {
	target := "2.20.19"
	store := InitConfigCreateTestStoreAt(&target)
	t.Cleanup(store.Close)
	p, err := createLegacyTestProject(store, db.Project{Name: "before upgrade"})
	require.NoError(t, err)
	wf := db.WorkflowTemplate{ProjectID: p.ID, Name: "approval"}
	require.NoError(t, store.Sql().Insert(&wf))
	rev := db.WorkflowRevision{ProjectID: p.ID, WorkflowTemplateID: wf.ID, Number: 1, Created: time.Now()}
	require.NoError(t, store.Sql().Insert(&rev))
	// Use the old schema explicitly: the current node model includes the new column.
	_, err = store.Sql().Exec("insert into project__workflow_node (workflow_template_id,revision_id,template_id,kind,convergence_mode,approval_message) values (?,?,0,'approval','all','Keep this')", wf.ID, rev.ID)
	require.NoError(t, err)
	run := db.WorkflowRun{ProjectID: p.ID, WorkflowTemplateID: wf.ID, RevisionID: rev.ID, Status: db.WorkflowRunApproval, Start: new(time.Now())}
	require.NoError(t, store.Sql().Insert(&run))
	require.NoError(t, db.Migrate(store, nil))
	graph, err := store.GetWorkflowRevisionGraph(p.ID, rev.ID)
	require.NoError(t, err)
	require.Len(t, graph.Nodes, 1)
	require.Nil(t, graph.Nodes[0].Notification)
	require.Equal(t, "Keep this", *graph.Nodes[0].ApprovalMessage)
	actual, err := store.GetWorkflowRunByID(p.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, db.WorkflowRunApproval, actual.Status)
	items, err := store.GetWorkflowNotifications(p.ID, run.ID)
	require.NoError(t, err)
	require.Empty(t, items)
}
