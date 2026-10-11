package sql

import (
	"encoding/json"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProjectNotificationMigrationAndIsolation(t *testing.T) {
	version := "2.20.21"
	store := InitConfigCreateTestStoreAt(&version)
	t.Cleanup(store.Close)
	project, err := createLegacyTestProject(store, db.Project{Name: "existing", Alert: true})
	require.NoError(t, err)
	other, err := createLegacyTestProject(store, db.Project{Name: "disabled"})
	require.NoError(t, err)
	legacy := `{"success":"Сохранено {{task.status}}","error":"Ошибка","timezone":"Europe/Moscow","locale":"ru"}`
	_, err = store.exec("update project set notification_templates=? where id in (?,?)", legacy, project.ID, other.ID)
	require.NoError(t, err)
	_, err = store.exec("insert into project__alert_channel(project_id,channel,enabled,settings,secret) values (?,'telegram',true,'{\"chat_id\":\"@saved\"}','encrypted-token')", project.ID)
	require.NoError(t, err)
	require.NoError(t, db.Migrate(store, nil))
	channel, err := store.GetAlertChannel(project.ID, "telegram")
	require.NoError(t, err)
	require.True(t, channel.Enabled)
	require.Equal(t, "encrypted-token", channel.Secret)
	require.JSONEq(t, `{"chat_id":"@saved"}`, channel.Settings)
	actual, err := taskmessage.Parse(channel.MessageTemplates)
	require.NoError(t, err)
	require.Equal(t, "Сохранено {{task.status}}", actual.Success)
	require.Equal(t, "ru", actual.SuccessLocale)
	disabled, err := store.GetAlertChannel(other.ID, "telegram")
	require.NoError(t, err)
	require.False(t, disabled.Enabled)
	require.Equal(t, channel.MessageTemplates, disabled.MessageTemplates)
	project.Name = "renamed"
	require.NoError(t, store.UpdateProject(project))
	all, err := store.GetProjectAlertTemplates(project.ID)
	require.NoError(t, err)
	require.Equal(t, *actual, all["telegram"])
	otherChannel, err := store.GetAlertChannel(project.ID, "slack")
	require.NoError(t, err)
	require.Nil(t, otherChannel.MessageTemplates)
	actual.Error = "{{ secrets }}"
	raw, err := json.Marshal(actual)
	require.NoError(t, err)
	channel.MessageTemplates = new(string(raw))
	require.Error(t, store.SetAlertChannel(channel))
	_, err = store.CreateProject(db.Project{Name: "invalid import", NotificationTemplates: new(`{}`)})
	require.Error(t, err)
	imported, err := store.CreateProject(db.Project{Name: "old backup", NotificationTemplates: &legacy})
	require.NoError(t, err)
	restored, err := store.GetProjectAlertTemplates(imported.ID)
	require.NoError(t, err)
	require.Equal(t, "Сохранено {{task.status}}", restored["telegram"].Success)
}
