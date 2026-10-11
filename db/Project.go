package db

import (
	"time"

	"github.com/impishMD/TExec/pkg/taskmessage"
)

// Project is the top level structure in TExec
type Project struct {
	ID               int       `db:"id" json:"id" backup:"-"`
	Name             string    `db:"name" json:"name" binding:"required"`
	Icon             *string   `db:"icon" json:"icon,omitempty"`
	Created          time.Time `db:"created" json:"created" backup:"-"`
	Alert            bool      `db:"alert" json:"alert,omitempty"`
	AlertChat        *string   `db:"alert_chat" json:"alert_chat,omitempty"`
	MaxParallelTasks int       `db:"max_parallel_tasks" json:"max_parallel_tasks,omitempty"`
	Type             string    `db:"type" json:"type"`
	// AlertTemplates transfers channel templates without credentials or destinations.
	AlertTemplates map[string]taskmessage.Settings `db:"-" json:"alert_templates,omitempty" backup:"alert_templates"`
	// NotificationTemplates reads backups made by the first local preview only.
	NotificationTemplates  *string `db:"-" json:"notification_templates,omitempty" backup:"notification_templates"`
	DefaultSecretStorageID *int    `db:"default_secret_storage_id" json:"default_secret_storage_id,omitempty" backup:"-"`
}
