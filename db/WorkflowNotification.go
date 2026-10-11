package db

import (
	"time"

	"github.com/impishMD/TExec/pkg/common_errors"
	"github.com/impishMD/TExec/pkg/workflowmessage"
)

type WorkflowNotificationConfig struct {
	Channel  string `json:"channel" backup:"channel"`
	Message  string `json:"message" backup:"message"`
	Timezone string `json:"timezone" backup:"timezone"`
	Locale   string `json:"locale,omitempty" backup:"locale"`
	// Nil uses all completed direct predecessors; explicit references must be ancestors.
	SourceNodeID *int   `json:"source_node_id,omitempty" backup:"source_node_id"`
	OnError      string `json:"on_error" backup:"on_error"`
}

func (c WorkflowNotificationConfig) Validate() error {
	if c.Channel != "telegram" {
		return common_errors.NewValidationError("unsupported workflow notification channel")
	}
	if c.OnError != "" && c.OnError != "continue" && c.OnError != "fail" {
		return common_errors.NewValidationError("notification failure policy must be continue or fail")
	}
	if err := workflowmessage.Validate(c.Message, c.Timezone); err != nil {
		return common_errors.NewValidationError(err.Error())
	}
	return nil
}

type WorkflowNotification struct {
	ID             int    `db:"id" json:"id"`
	ProjectID      int    `db:"project_id" json:"project_id"`
	WorkflowRunID  int    `db:"workflow_run_id" json:"workflow_run_id"`
	WorkflowNodeID int    `db:"workflow_node_id" json:"workflow_node_id"`
	Status         string `db:"status" json:"status"`
	Attempts       int    `db:"attempts" json:"attempts"`
	// The rendered message is frozen for retries, but not exposed by the run API.
	Message       string     `db:"message" json:"-"`
	LastError     string     `db:"last_error" json:"last_error,omitempty"`
	Created       time.Time  `db:"created" json:"created"`
	NextAttemptAt time.Time  `db:"next_attempt_at" json:"next_attempt_at"`
	Resolved      *time.Time `db:"resolved" json:"resolved,omitempty"`
}

func (n WorkflowNotification) Pending() bool { return n.Status == "queued" || n.Status == "sending" }
