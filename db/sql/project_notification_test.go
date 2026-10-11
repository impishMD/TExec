package sql

import (
	"testing"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/taskmessage"
	"github.com/stretchr/testify/require"
)

func TestProjectNotificationMigrationAndIsolation(t *testing.T) {
	version := "2.20.20"
	store := InitConfigCreateTestStoreAt(&version)
	t.Cleanup(store.Close)
	project, err := createLegacyTestProject(store, db.Project{Name: "existing", Alert: true})
	require.NoError(t, err)
	require.NoError(t, db.Migrate(store, nil))
	project, err = store.GetProject(project.ID)
	require.NoError(t, err)
	require.True(t, project.Alert)
	require.Nil(t, project.NotificationTemplates)
	settings := taskmessage.Defaults("ru")
	require.NoError(t, store.SetProjectNotificationTemplates(project.ID, settings))
	// An older/stale general settings form must not remove notification templates.
	project.Name = "renamed"
	require.NoError(t, store.UpdateProject(project))
	project, err = store.GetProject(project.ID)
	require.NoError(t, err)
	actual, err := taskmessage.Parse(project.NotificationTemplates)
	require.NoError(t, err)
	require.Equal(t, settings, *actual)
	other, err := store.CreateProject(db.Project{Name: "other"})
	require.NoError(t, err)
	require.Nil(t, other.NotificationTemplates)
	settings.Error = "{{ secrets }}"
	require.Error(t, store.SetProjectNotificationTemplates(project.ID, settings))
	_, err = store.CreateProject(db.Project{Name: "invalid import", NotificationTemplates: new(`{}`)})
	require.Error(t, err)
}
