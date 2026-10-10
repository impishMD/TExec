package sql

import (
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/tz"
)

func (d *SqlDb) GetWorkflowNotifications(p, runID int) ([]db.WorkflowNotification, error) {
	items := []db.WorkflowNotification{}
	_, err := d.selectAll(&items, "select * from project__workflow_notification where project_id=? and workflow_run_id=? order by id", p, runID)
	return items, err
}

func (d *SqlDb) CreateWorkflowNotification(item db.WorkflowNotification) (db.WorkflowNotification, error) {
	var count int
	err := d.selectOne(&count, "select count(*) from project__workflow_run r join project__workflow_node n on n.revision_id=r.revision_id where r.project_id=? and r.id=? and n.id=? and n.kind in ('notification','approval')", item.ProjectID, item.WorkflowRunID, item.WorkflowNodeID)
	if err != nil {
		return item, err
	}
	if count != 1 {
		return item, db.ErrNotFound
	}
	item.ID, item.Attempts, item.Status = 0, 0, "queued"
	item.Created, item.NextAttemptAt, item.Resolved = tz.Now(), tz.Now(), nil
	err = d.Sql().Insert(&item)
	return item, err
}

func (d *SqlDb) GetDueWorkflowNotifications() ([]db.WorkflowNotification, error) {
	items := []db.WorkflowNotification{}
	_, err := d.selectAll(&items, "select * from project__workflow_notification where status in ('queued','sending') and next_attempt_at<=? order by next_attempt_at,id limit 20", tz.Now())
	return items, err
}

// A lease recovers a worker interrupted during delivery; the attempt counter also
// bounds retries after crashes. Telegram has no exactly-once delivery guarantee.
func (d *SqlDb) ClaimWorkflowNotification(item db.WorkflowNotification) (bool, error) {
	result, err := d.exec("update project__workflow_notification set status='sending',attempts=attempts+1,next_attempt_at=? where project_id=? and id=? and status in ('queued','sending') and attempts=? and next_attempt_at<=?", tz.Now().Add(time.Minute), item.ProjectID, item.ID, item.Attempts, tz.Now())
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	return count == 1, err
}

func (d *SqlDb) UpdateWorkflowNotification(item db.WorkflowNotification) error {
	_, err := d.exec("update project__workflow_notification set status=?,last_error=?,next_attempt_at=?,resolved=? where project_id=? and id=? and status in ('queued','sending') and attempts=?", item.Status, item.LastError, item.NextAttemptAt, item.Resolved, item.ProjectID, item.ID, item.Attempts)
	return err
}

func (d *SqlDb) CancelWorkflowNotifications(p, runID int) error {
	_, err := d.exec("update project__workflow_notification set status='cancelled',resolved=? where project_id=? and workflow_run_id=? and status in ('queued','sending')", tz.Now(), p, runID)
	return err
}

func (d *SqlDb) GetWorkflowTaskOutputTail(p, taskID int) ([]string, error) {
	if err := d.validateTask(p, taskID); err != nil {
		return nil, err
	}
	// Cap both record count and individual record size before loading logs.
	var rows []struct {
		Output string `db:"output"`
	}
	_, err := d.selectAll(&rows, "select substr(output,1,4096) as output from task__output where task_id=? order by time desc,id desc limit 100", taskID)
	result := make([]string, len(rows))
	for i, row := range rows {
		result[len(rows)-1-i] = row.Output
	}
	return result, err
}
