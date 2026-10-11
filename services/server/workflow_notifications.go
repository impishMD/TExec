package server

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/task_logger"
	"github.com/impishMD/TExec/pkg/tz"
	"github.com/impishMD/TExec/pkg/workflowmessage"
	"github.com/impishMD/TExec/services/audit"
	"github.com/impishMD/TExec/util"
)

type WorkflowNotificationSender func(context.Context, int, string, string, string) error

func WorkflowRunURL(projectID, workflowID, runID int) string {
	base, err := url.Parse(util.Config.WebHost)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return ""
	}
	base.Path = strings.TrimRight(base.Path, "/") + fmt.Sprintf("/project/%d/workflows/%d/runs/%d", projectID, workflowID, runID)
	base.RawQuery, base.Fragment, base.User = "", "", nil
	return base.String()
}

func notificationText(locale, english, russian string) string {
	if strings.HasPrefix(locale, "ru") {
		return russian
	}
	return english
}

func notificationStatus(locale, status string) string {
	if !strings.HasPrefix(locale, "ru") {
		return status
	}
	labels := map[string]string{"success": "Успешно", "error": "Ошибка", "failed": "Ошибка", "stopped": "Остановлено", "approved": "Подтверждено", "rejected": "Отклонено", "sent": "Отправлено", "cancelled": "Отменено"}
	if label := labels[status]; label != "" {
		return label
	}
	return status
}

func (s *workflowService) queueNotification(graph db.WorkflowTemplate, run db.WorkflowRun, node db.WorkflowNode, approvals map[int]db.WorkflowApproval, tasks map[int]db.Task, delays map[int]db.WorkflowDelay, notifications map[int]db.WorkflowNotification, initiator *db.User) error {
	config := node.Notification
	if config == nil {
		return nil
	}
	if _, exists := notifications[node.ID]; exists {
		return nil
	}
	project, err := s.store.GetProject(run.ProjectID)
	if err != nil {
		return err
	}
	values := map[string]string{
		"project.name": project.Name, "workflow.name": graph.Name, "run.id": strconv.Itoa(run.ID),
		"run.url":        WorkflowRunURL(run.ProjectID, graph.ID, run.ID),
		"run.started_by": notificationText(config.Locale, "System", "Система"),
	}
	if initiator != nil {
		values["run.started_by"] = initiator.Username
	}
	at := tz.Now()
	if approval, ok := approvals[node.ID]; ok {
		at = approval.Created
		values["approval.deadline"] = notificationText(config.Locale, "No deadline", "Без ограничения")
		if node.ApprovalTimeout != nil {
			location, _ := time.LoadLocation(config.Timezone)
			if location == nil {
				location = time.UTC
			}
			values["approval.deadline"] = approval.Created.Add(time.Duration(*node.ApprovalTimeout) * time.Second).In(location).Format("02.01.2006 15:04:05 MST")
		}
	}
	selected := map[int]bool{}
	if config.SourceNodeID != nil {
		selected[*config.SourceNodeID] = true
	} else {
		for _, edge := range graph.Edges {
			if edge.DestinationNodeID == node.ID {
				selected[edge.SourceNodeID] = true
			}
		}
	}
	var names, statuses, diagnostics []string
	for _, source := range graph.Nodes {
		if !selected[source.ID] {
			continue
		}
		name, status, diagnostic := "", "", ""
		if task, ok := tasks[source.ID]; ok && task.Status.IsFinished() && task.End != nil {
			template, e := s.store.GetTemplate(run.ProjectID, source.TemplateID)
			if e != nil {
				return e
			}
			name, status = template.Name, string(task.Status)
			if task.Status != task_logger.TaskSuccessStatus && strings.Contains(config.Message, "source.error") {
				outputs, e := s.repo.GetWorkflowTaskOutputTail(run.ProjectID, task.ID)
				if e != nil {
					return e
				}
				diagnostic = workflowmessage.ErrorSummary(outputs)
				if diagnostic == "" {
					diagnostic = notificationText(config.Locale, "No error summary; see task log", "Описание ошибки не найдено; откройте журнал задачи")
				}
			}
		} else if approval, ok := approvals[source.ID]; ok && approval.Status != db.WorkflowApprovalPending {
			name, status = notificationText(config.Locale, "Approval", "Подтверждение"), string(approval.Status)
		} else if delay, ok := delays[source.ID]; ok && delay.Status != db.WorkflowDelayWaiting {
			name, status = notificationText(config.Locale, "Delay", "Задержка"), string(delay.Status)
		} else if delivery, ok := notifications[source.ID]; ok && !delivery.Pending() {
			name, status, diagnostic = notificationText(config.Locale, "Notification", "Оповещение"), delivery.Status, delivery.LastError
		}
		if status == "" {
			continue
		}
		names, statuses = append(names, name), append(statuses, notificationStatus(config.Locale, status))
		if diagnostic != "" {
			diagnostics = append(diagnostics, name+": "+diagnostic)
		}
	}
	values["source.name"], values["source.status"], values["source.error"] = strings.Join(names, ", "), strings.Join(statuses, ", "), workflowmessage.Truncate(strings.Join(diagnostics, "\n"), 1500)
	message, err := workflowmessage.Render(config.Message, config.Timezone, at, values)
	if err != nil {
		return err
	}
	_, err = s.repo.CreateWorkflowNotification(db.WorkflowNotification{ProjectID: run.ProjectID, WorkflowRunID: run.ID, WorkflowNodeID: node.ID, Message: message})
	return err
}

// ProcessWorkflowNotifications sends outside the graph progression loop. Every
// delivery is leased in SQL and serialized with approval/stop operations per run.
func (s *workflowService) ProcessWorkflowNotifications(ctx context.Context) error {
	items, err := s.repo.GetDueWorkflowNotifications()
	if err != nil {
		return err
	}
	for _, item := range items {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		unlock := s.lock(item.WorkflowRunID)
		err = s.deliverNotification(ctx, item)
		unlock()
		if err != nil {
			return err
		}
		if err = s.ProgressWorkflowRun(item.ProjectID, item.WorkflowRunID, nil); err != nil && !errors.Is(err, db.ErrNotFound) {
			return err
		}
	}
	return nil
}

func (s *workflowService) deliverNotification(ctx context.Context, item db.WorkflowNotification) error {
	run, err := s.repo.GetWorkflowRunByID(item.ProjectID, item.WorkflowRunID)
	if errors.Is(err, db.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if run.Status.IsFinished() {
		return s.repo.CancelWorkflowNotifications(run.ProjectID, run.ID)
	}
	graph, err := s.repo.GetWorkflowRevisionGraph(run.ProjectID, run.RevisionID)
	if err != nil {
		return err
	}
	var node db.WorkflowNode
	for _, n := range graph.Nodes {
		if n.ID == item.WorkflowNodeID {
			node = n
			break
		}
	}
	if node.Notification == nil {
		return db.ErrInvalidOperation
	}
	if node.EffectiveKind() == db.WorkflowNodeApprovalKind {
		approval, e := s.repo.GetWorkflowApproval(run.ProjectID, run.ID, node.ID)
		if e != nil {
			return e
		}
		expired := node.ApprovalTimeout != nil && !tz.Now().Before(approval.Created.Add(time.Duration(*node.ApprovalTimeout)*time.Second))
		if approval.Status != db.WorkflowApprovalPending || expired {
			item.Status, item.Resolved = "cancelled", new(tz.Now())
			return s.repo.UpdateWorkflowNotification(item)
		}
	}
	if item.Attempts >= 3 {
		item.Status, item.Resolved, item.LastError = "failed", new(tz.Now()), "Notification delivery attempts exhausted"
		return s.repo.UpdateWorkflowNotification(item)
	}
	claimed, err := s.repo.ClaimWorkflowNotification(item)
	if err != nil || !claimed {
		return err
	}
	item.Attempts++
	button := notificationText(node.Notification.Locale, "Open run", "Открыть запуск")
	if node.EffectiveKind() == db.WorkflowNodeApprovalKind {
		button = notificationText(node.Notification.Locale, "Open approval", "Открыть подтверждение")
	}
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = s.sendNotification(sendCtx, run.ProjectID, item.Message, WorkflowRunURL(run.ProjectID, graph.ID, run.ID), button)
	cancel()
	item.LastError, item.Status, item.Resolved = "", "sent", new(tz.Now())
	if err != nil {
		// The alerting transport returns only credential-free diagnostics.
		item.LastError = workflowmessage.Truncate(err.Error(), 500)
		item.Status = "failed"
		if item.Attempts < 3 {
			item.Status, item.Resolved = "queued", nil
			item.NextAttemptAt = tz.Now().Add(time.Duration(item.Attempts*10) * time.Second)
		}
	}
	if err = s.repo.UpdateWorkflowNotification(item); err != nil {
		return err
	}
	if item.Status == "sent" {
		s.record(audit.WorkflowNotificationSent, run, node.ID, nil)
	}
	if item.Status == "failed" {
		s.record(audit.WorkflowNotificationFailed, run, node.ID, nil)
	}
	return nil
}
