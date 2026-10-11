package sql

import (
	"testing"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/pkg/task_logger"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/require"
)

func TestTaskStatsIncludesFailuresBeforeStart(t *testing.T) {
	previous := util.Config
	store := InitConfigCreateTestStore()
	t.Cleanup(func() { store.Close(); util.Config = previous })
	projectID, repositoryID := newTemplateTestProject(t, store)
	tpl, err := store.CreateTemplate(db.Template{ProjectID: projectID, RepositoryID: repositoryID, Name: "stats", Playbook: "site.yml"})
	require.NoError(t, err)
	other, err := store.CreateTemplate(db.Template{ProjectID: projectID, RepositoryID: repositoryID, Name: "other", Playbook: "site.yml"})
	require.NoError(t, err)
	user, err := store.CreateUserWithoutPassword(db.User{Username: "stats", Name: "Stats", Email: "stats@example.test"})
	require.NoError(t, err)
	start := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 2)
	day := start.AddDate(0, 0, 1)
	add := func(templateID int, status task_logger.TaskStatus, created time.Time, started *time.Time, userID *int) {
		t.Helper()
		_, err := store.CreateTask(db.Task{ProjectID: projectID, TemplateID: templateID, Status: status, Created: created, Start: started, UserID: userID, Playbook: "site.yml"}, 0)
		require.NoError(t, err)
	}
	add(tpl.ID, "success", start, &start, &user.ID)
	for range 3 {
		add(tpl.ID, "success", day, &day, &user.ID)
	}
	add(tpl.ID, "error", day, &day, &user.ID)
	add(tpl.ID, "error", day, nil, &user.ID)                    // Failed before a runner started it.
	add(tpl.ID, "stopped", day, nil, nil)                       // Canceled while queued.
	add(tpl.ID, "error", start.Add(-time.Hour), &day, &user.ID) // Outside the creation-date range.
	add(tpl.ID, "error", end, nil, &user.ID)                    // Exclusive end date.
	add(other.ID, "error", day, nil, &user.ID)
	filter := db.TaskFilter{Start: &start, End: &end}
	stats, err := store.GetTaskStats(projectID, &tpl.ID, db.TaskStatUnitDay, filter)
	require.NoError(t, err)
	require.Equal(t, []db.TaskStat{
		{Date: "2026-10-10", CountByStatus: map[task_logger.TaskStatus]int{"success": 1}},
		{Date: "2026-10-11", CountByStatus: map[task_logger.TaskStatus]int{"success": 3, "error": 2, "stopped": 1}},
	}, stats)
	filter.UserID = &user.ID
	stats, err = store.GetTaskStats(projectID, &tpl.ID, db.TaskStatUnitDay, filter)
	require.NoError(t, err)
	require.Equal(t, map[task_logger.TaskStatus]int{"success": 3, "error": 2}, stats[1].CountByStatus)
	filter.UserID = nil
	stats, err = store.GetTaskStats(projectID, nil, db.TaskStatUnitDay, filter)
	require.NoError(t, err)
	require.Equal(t, 3, stats[1].CountByStatus["error"])
	stats, err = store.GetTaskStats(projectID+100, nil, db.TaskStatUnitDay, filter)
	require.NoError(t, err)
	require.Empty(t, stats)
}
