package tasks

import (
	"strconv"
	"strings"

	"github.com/impishMD/TExec/pkg/notificationmessage"
	"github.com/impishMD/TExec/pkg/task_logger"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/pkg/tz"
)

// Render with the current settings of this channel at completion.
// Empty means legacy formatting, including all non-terminal status notifications.
func (t *TaskRunner) projectAlertMessage(channel string) string {
	if t.shouldSkipStatusAlert() || (t.Task.Status != task_logger.TaskSuccessStatus && t.Task.Status != task_logger.TaskFailStatus) {
		return ""
	}
	saved, err := t.pool.store.GetAlertChannel(t.Template.ProjectID, channel)
	if err != nil {
		t.Log("Can't load project notification templates; using default message")
		return ""
	}
	settings, err := taskmessage.Parse(saved.MessageTemplates)
	if err != nil {
		t.Log("Invalid project notification templates; using default message")
		return ""
	}
	if settings == nil {
		return ""
	}
	project, err := t.pool.store.GetProject(t.Template.ProjectID)
	if err != nil {
		t.Log("Can't load project for notification; using default message")
		return ""
	}
	at := tz.Now()
	if t.Task.End != nil {
		at = *t.Task.End
	}
	author, version := t.alertInfos()
	if t.Task.ProjectTokenName != "" {
		author = t.Task.ProjectTokenName
	}
	values := map[string]string{
		"project.name": project.Name, "template.name": t.Template.Name,
		"task.id": strconv.Itoa(t.Task.ID), "task.started_by": author,
		"task.url": t.taskLink(), "task.version": version, "task.message": t.Task.Message,
	}
	if t.Task.Status == task_logger.TaskFailStatus && strings.Contains(settings.Error, "task.error") && t.pool.workflowRepo != nil {
		// Task output is buffered, including remote runner output. Flush before
		// reading, without finalizing the Ansible summary ahead of task completion.
		t.pool.syncLogs(nil, at)
		if output, err := t.pool.workflowRepo.GetWorkflowTaskOutputTail(project.ID, t.Task.ID); err == nil {
			values["task.error"] = notificationmessage.ErrorSummary(output)
		}
	}
	message, err := settings.Render(string(t.Task.Status), at, values)
	if err != nil {
		t.Log("Can't render project notification; using default message")
		return ""
	}
	return message
}
