package server

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/task_logger"
	"github.com/impishMD/TExec/pkg/tz"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/require"
)

func notificationConfig() *db.WorkflowNotificationConfig {
	return &db.WorkflowNotificationConfig{Channel: "telegram", Message: "{{ project.name }}: {{ workflow.name }} #{{ run.id }}\n{{ source.name }}: {{ source.error }}", Timezone: "Europe/Moscow", Locale: "ru", OnError: "continue"}
}

func TestWorkflowNotificationSeesPredecessorResolvedInSameProgression(t *testing.T) {
	for _, kind := range []db.WorkflowNodeKind{db.WorkflowNodeDelayKind, db.WorkflowNodeApprovalKind} {
		t.Run(string(kind), func(t *testing.T) {
			store, p, _, _, svc := workflowServiceFixture(t)
			config := notificationConfig()
			config.Message = "{{ source.name }}: {{ source.status }}"
			graph, err := store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: p.ID, Name: "resolved source", Nodes: []db.WorkflowNode{
				{ID: 1, Kind: kind, DelaySeconds: new(1), ApprovalTimeout: new(1)},
				{ID: 2, Kind: db.WorkflowNodeNotificationKind, Notification: config},
			}, Edges: []db.WorkflowEdge{{SourceNodeID: 1, DestinationNodeID: 2, Condition: db.WorkflowEdgeAlways}}})
			require.NoError(t, err)
			run, err := svc.StartWorkflow(graph, nil)
			require.NoError(t, err)
			if kind == db.WorkflowNodeDelayKind {
				_, err = store.Sql().Exec("update project__workflow_delay set resume_at=? where workflow_run_id=?", tz.Now().Add(-time.Minute), run.ID)
			} else {
				_, err = store.Sql().Exec("update project__workflow_approval set created=? where workflow_run_id=?", tz.Now().Add(-time.Minute), run.ID)
			}
			require.NoError(t, err)
			require.NoError(t, svc.ProgressWorkflowRun(p.ID, run.ID, nil))
			items, err := store.GetWorkflowNotifications(p.ID, run.ID)
			require.NoError(t, err)
			require.Len(t, items, 1)
			if kind == db.WorkflowNodeDelayKind {
				require.Equal(t, "Задержка: Успешно", items[0].Message)
			} else {
				require.Equal(t, "Подтверждение: Отклонено", items[0].Message)
			}
		})
	}
}

func TestWorkflowNotificationQueuedOnceAndSurvivesRestart(t *testing.T) {
	store, p, tpl, queue, svc := workflowServiceFixture(t)
	graph, err := store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: p.ID, Name: "notify", Nodes: []db.WorkflowNode{
		{ID: 1, Kind: db.WorkflowNodeNotificationKind, Notification: notificationConfig()}, {ID: 2, TemplateID: tpl.ID},
	}, Edges: []db.WorkflowEdge{{SourceNodeID: 1, DestinationNodeID: 2, Condition: db.WorkflowEdgeOnSuccess}}})
	require.NoError(t, err)
	run, err := svc.StartWorkflow(graph, nil)
	require.NoError(t, err)
	require.Empty(t, workflowTasks(t, store, run), "wait for delivery before starting next task")
	var calls atomic.Int32
	svc = NewWorkflowService(store, store, queue, nil, func(_ context.Context, project int, text, link, button string) error {
		calls.Add(1)
		require.Equal(t, p.ID, project)
		require.Contains(t, text, "notify #")
		return nil
	})
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Go(func() { results <- svc.ProcessWorkflowNotifications(context.Background()) })
	}
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	require.Equal(t, int32(1), calls.Load())
	items, err := store.GetWorkflowNotifications(p.ID, run.ID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "sent", items[0].Status)
	require.Len(t, workflowTasks(t, store, run), 1)
	require.NoError(t, svc.ProgressWorkflowRun(p.ID, run.ID, nil))
	require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
	require.Equal(t, int32(1), calls.Load())
}

func TestWorkflowNotificationFailurePoliciesAndRetryBudget(t *testing.T) {
	for _, policy := range []string{"continue", "fail"} {
		t.Run(policy, func(t *testing.T) {
			store, p, _, queue, _ := workflowServiceFixture(t)
			calls := 0
			svc := NewWorkflowService(store, store, queue, nil, func(context.Context, int, string, string, string) error {
				calls++
				return errors.New("network failure")
			})
			config := notificationConfig()
			config.OnError = policy
			graph, err := store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: p.ID, Name: "retry", Nodes: []db.WorkflowNode{{ID: 1, Kind: db.WorkflowNodeNotificationKind, Notification: config}}})
			require.NoError(t, err)
			run, err := svc.StartWorkflow(graph, nil)
			require.NoError(t, err)
			for i := 0; i < 3; i++ {
				_, err = store.Sql().Exec("update project__workflow_notification set next_attempt_at=? where workflow_run_id=?", tz.Now().Add(-time.Minute), run.ID)
				require.NoError(t, err)
				require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
			}
			require.Equal(t, 3, calls)
			require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
			require.Equal(t, 3, calls)
			items, err := store.GetWorkflowNotifications(p.ID, run.ID)
			require.NoError(t, err)
			require.Equal(t, "failed", items[0].Status)
			require.Equal(t, 3, items[0].Attempts)
			run, err = store.GetWorkflowRunByID(p.ID, run.ID)
			require.NoError(t, err)
			expected := db.WorkflowRunSuccess
			if policy == "fail" {
				expected = db.WorkflowRunFailed
			}
			require.Equal(t, expected, run.Status)
		})
	}
}

func TestApprovalNotificationIsReconciledButNeverChangesApproval(t *testing.T) {
	store, p, _, queue, _ := workflowServiceFixture(t)
	svc := NewWorkflowService(store, store, queue, nil, func(context.Context, int, string, string, string) error { return errors.New("offline") })
	config := notificationConfig()
	config.OnError = "fail"
	graph, err := store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: p.ID, Name: "approval", Nodes: []db.WorkflowNode{{ID: 1, Kind: db.WorkflowNodeApprovalKind, Notification: config}}})
	require.NoError(t, err)
	run, err := svc.StartWorkflow(graph, nil)
	require.NoError(t, err)
	// Simulate a restart after persisting approval but before adding the outbox row.
	_, err = store.Sql().Exec("delete from project__workflow_notification where workflow_run_id=?", run.ID)
	require.NoError(t, err)
	require.NoError(t, svc.ProgressWorkflowRun(p.ID, run.ID, nil))
	for i := 0; i < 3; i++ {
		_, err = store.Sql().Exec("update project__workflow_notification set next_attempt_at=?", tz.Now().Add(-time.Minute))
		require.NoError(t, err)
		require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
	}
	approval, err := store.GetWorkflowApproval(p.ID, run.ID, graph.Nodes[0].ID)
	require.NoError(t, err)
	require.Equal(t, db.WorkflowApprovalPending, approval.Status)
	run, err = store.GetWorkflowRunByID(p.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, db.WorkflowRunApproval, run.Status)
	_, err = svc.ResolveWorkflowApproval(p.ID, graph.ID, run.ID, graph.Nodes[0].ID, db.WorkflowApprovalApproved, nil)
	require.NoError(t, err)
}

func TestWorkflowNotificationCancelledAfterStopResolutionOrTimeout(t *testing.T) {
	for _, action := range []string{"stop", "approve", "timeout"} {
		t.Run(action, func(t *testing.T) {
			store, p, _, queue, _ := workflowServiceFixture(t)
			calls := 0
			svc := NewWorkflowService(store, store, queue, nil, func(context.Context, int, string, string, string) error { calls++; return nil })
			graph, err := store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: p.ID, Name: "approval", Nodes: []db.WorkflowNode{{ID: 1, Kind: db.WorkflowNodeApprovalKind, ApprovalTimeout: new(60), Notification: notificationConfig()}}})
			require.NoError(t, err)
			run, err := svc.StartWorkflow(graph, nil)
			require.NoError(t, err)
			switch action {
			case "stop":
				_, err = svc.StopWorkflowRun(p.ID, run.ID, nil)
			case "approve":
				_, err = svc.ResolveWorkflowApproval(p.ID, graph.ID, run.ID, graph.Nodes[0].ID, db.WorkflowApprovalApproved, nil)
			case "timeout":
				_, err = store.Sql().Exec("update project__workflow_approval set created=? where workflow_run_id=?", tz.Now().Add(-time.Hour), run.ID)
			}
			require.NoError(t, err)
			require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
			require.Equal(t, 0, calls)
			items, err := store.GetWorkflowNotifications(p.ID, run.ID)
			require.NoError(t, err)
			require.Equal(t, "cancelled", items[0].Status)
		})
	}
}

func TestWorkflowNotificationSourceMappingErrorExcerptAndPinnedRevision(t *testing.T) {
	store, p, tpl, queue, _ := workflowServiceFixture(t)
	var message string
	svc := NewWorkflowService(store, store, queue, nil, func(_ context.Context, _ int, text, _, _ string) error { message = text; return nil })
	config := notificationConfig()
	config.SourceNodeID = new(-1)
	graph, err := store.CreateWorkflowTemplate(db.WorkflowTemplate{ProjectID: p.ID, Name: "original", Nodes: []db.WorkflowNode{
		{ID: -1, TemplateID: tpl.ID}, {ID: -2, Kind: db.WorkflowNodeNotificationKind, Notification: config},
	}, Edges: []db.WorkflowEdge{{SourceNodeID: -1, DestinationNodeID: -2, Condition: db.WorkflowEdgeOnFailure}}})
	require.NoError(t, err)
	require.Equal(t, graph.Nodes[0].ID, *graph.Nodes[1].Notification.SourceNodeID)
	run, err := svc.StartWorkflow(graph, nil)
	require.NoError(t, err)
	task := workflowTasks(t, store, run)[0].Task
	_, err = store.CreateTaskOutput(db.TaskOutput{TaskID: task.ID, Time: tz.Now(), Output: "[ERROR]: Connection refused password=hidden-secret"})
	require.NoError(t, err)
	finishWorkflowTask(t, store, svc, task, task_logger.TaskFailStatus)
	graph.Nodes[1].Notification.Message = "edited message"
	require.NoError(t, store.UpdateWorkflowTemplate(graph))
	latest, err := store.GetWorkflowTemplate(p.ID, graph.ID)
	require.NoError(t, err)
	require.NotEqual(t, graph.Nodes[0].ID, latest.Nodes[0].ID)
	require.Equal(t, latest.Nodes[0].ID, *latest.Nodes[1].Notification.SourceNodeID)
	require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
	require.Contains(t, message, "Connection refused")
	require.Contains(t, message, "shell")
	require.NotContains(t, message, "hidden-secret")
	require.NotContains(t, message, "edited message")
}

func TestWorkflowNotificationValidationAndLeaseRecovery(t *testing.T) {
	store, p, _, queue, svc := workflowServiceFixture(t)
	config := notificationConfig()
	config.SourceNodeID = new(999)
	graph := db.WorkflowTemplate{ProjectID: p.ID, Name: "test", Nodes: []db.WorkflowNode{{ID: 1, Kind: db.WorkflowNodeNotificationKind, Notification: config}}}
	_, err := store.CreateWorkflowTemplate(graph)
	require.Error(t, err)
	config.SourceNodeID = nil
	config.Message = "{{ password }}"
	_, err = store.CreateWorkflowTemplate(graph)
	require.Error(t, err)
	config.Message = "hello"
	graph, err = store.CreateWorkflowTemplate(graph)
	require.NoError(t, err)
	run, err := svc.StartWorkflow(graph, nil)
	require.NoError(t, err)
	items, err := store.GetDueWorkflowNotifications()
	require.NoError(t, err)
	require.Len(t, items, 1)
	claimed, err := store.ClaimWorkflowNotification(items[0])
	require.NoError(t, err)
	require.True(t, claimed)
	calls := 0
	svc = NewWorkflowService(store, store, queue, nil, func(context.Context, int, string, string, string) error { calls++; return nil })
	require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
	require.Equal(t, 0, calls)
	_, err = store.Sql().Exec("update project__workflow_notification set next_attempt_at=?", tz.Now().Add(-time.Minute))
	require.NoError(t, err)
	require.NoError(t, svc.ProcessWorkflowNotifications(context.Background()))
	require.Equal(t, 1, calls)
	items, err = store.GetWorkflowNotifications(p.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, 2, items[0].Attempts)
	old := util.Config.WebHost
	t.Cleanup(func() { util.Config.WebHost = old })
	util.Config.WebHost = "https://texec.example.com/prefix/"
	require.True(t, strings.HasPrefix(WorkflowRunURL(p.ID, graph.ID, run.ID), "https://texec.example.com/prefix/project/"))
}
