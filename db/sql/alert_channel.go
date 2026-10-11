package sql

import (
	"errors"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/impishMD/TExec/util"
)

func (d *SqlDb) GetAlertChannel(projectID int, channel string) (res db.AlertChannel, err error) {
	res = db.AlertChannel{ProjectID: projectID, Channel: channel, Settings: "{}"}
	if projectID == 0 {
		err = d.selectOne(&res, "select 0 as project_id, channel, enabled, settings, secret from alert_channel where channel=?", channel)
	} else {
		err = d.selectOne(&res, "select * from project__alert_channel where project_id=? and channel=?", projectID, channel)
	}
	if errors.Is(err, db.ErrNotFound) {
		err = nil
	}
	return
}

func (d *SqlDb) SetAlertChannel(c db.AlertChannel) error {
	if _, err := taskmessage.Parse(c.MessageTemplates); err != nil {
		return err
	}
	query := "insert into alert_channel (channel, enabled, settings, secret) values (?, ?, ?, ?)"
	args := []any{c.Channel, c.Enabled, c.Settings, c.Secret}
	conflict := "channel"
	if c.ProjectID != 0 {
		query = "insert into project__alert_channel (project_id, channel, enabled, settings, secret, message_templates) values (?, ?, ?, ?, ?, ?)"
		args = append(append([]any{c.ProjectID}, args...), c.MessageTemplates)
		conflict = "project_id, channel"
	}
	if d.GetDialect() == util.DbDriverMySQL {
		query += " on duplicate key update enabled=values(enabled), settings=values(settings), secret=values(secret)"
	} else {
		query += " on conflict (" + conflict + ") do update set enabled=excluded.enabled, settings=excluded.settings, secret=excluded.secret"
	}
	if c.ProjectID != 0 {
		if d.GetDialect() == util.DbDriverMySQL {
			query += ", message_templates=values(message_templates)"
		} else {
			query += ", message_templates=excluded.message_templates"
		}
	}
	_, err := d.exec(query, args...)
	return err
}

func (d *SqlDb) GetAlertChannelsWithSecrets() (res []db.AlertChannel, err error) {
	_, err = d.selectAll(&res, "select 0 as project_id, channel, enabled, settings, secret, null as message_templates from alert_channel where secret<>'' union all select project_id, channel, enabled, settings, secret, message_templates from project__alert_channel where secret<>''")
	return
}

// Returns only message templates, never channel credentials or destinations.
func (d *SqlDb) GetProjectAlertTemplates(projectID int) (map[string]taskmessage.Settings, error) {
	var channels []db.AlertChannel
	_, err := d.selectAll(&channels, "select channel, message_templates from project__alert_channel where project_id=? and message_templates is not null", projectID)
	if err != nil {
		return nil, err
	}
	result := map[string]taskmessage.Settings{}
	for _, channel := range channels {
		settings, err := taskmessage.Parse(channel.MessageTemplates)
		if err != nil {
			return nil, err
		}
		result[channel.Channel] = *settings
	}
	return result, nil
}
