package schedules

import (
	"sync"
	"testing"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/db/sql"
	"github.com/impishMD/TExec/pkg/ssh"
	"github.com/impishMD/TExec/pkg/task_logger"
	"github.com/impishMD/TExec/services/server"
	"github.com/impishMD/TExec/services/tasks"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEncryptionService is a test implementation of AccessKeyEncryptionService
type mockEncryptionService struct{}

func (m *mockEncryptionService) RekeyAccessKeys(oldKey string) (err error) {
	return nil
}

func (m *mockEncryptionService) SerializeSecret(key *db.AccessKey) error {
	return nil
}

func (m *mockEncryptionService) DeserializeSecret(key *db.AccessKey) error {
	return nil
}

func (m *mockEncryptionService) FillEnvironmentSecrets(env *db.Environment, deserializeSecret bool) error {
	return nil
}

func (m *mockEncryptionService) DeleteSecret(key *db.AccessKey) error {
	return nil
}

func (m *mockEncryptionService) CreateTaskSurveySecrets(projectID int, taskID int, secrets string, expireAt time.Time) error {
	return nil
}

func (m *mockEncryptionService) GetTaskSurveySecrets(projectID int, taskID int) (string, error) {
	return "", nil
}

func (m *mockEncryptionService) DeleteTaskSurveySecrets(projectID int, taskID int) error {
	return nil
}

func TestValidateCronFormat(t *testing.T) {
	err := ValidateCronFormat("* * * *")
	if err == nil {
		t.Fatal("")
	}

	err = ValidateCronFormat("* * 1 * *")
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestOneTimeSchedule(t *testing.T) {
	future := time.Now().Add(time.Hour)
	schedule := oneTimeSchedule{runAt: future}

	if schedule.Next(time.Now()) != future {
		t.Fatalf("expected next run at %v", future)
	}

	if !schedule.Next(future).IsZero() {
		t.Fatalf("expected schedule to stop after run time")
	}
}

// mockDeduplicator is a test implementation of ScheduleDeduplicator
type mockDeduplicator struct {
	mu             sync.Mutex
	allowExecution map[int]bool
	lockAttempts   map[int]int
}

func newMockDeduplicator() *mockDeduplicator {
	return &mockDeduplicator{
		allowExecution: make(map[int]bool),
		lockAttempts:   make(map[int]int),
	}
}

func (m *mockDeduplicator) TryLockExecution(scheduleID int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lockAttempts[scheduleID]++
	return m.allowExecution[scheduleID]
}

func (m *mockDeduplicator) setAllowExecution(scheduleID int, allow bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowExecution[scheduleID] = allow
}

func (m *mockDeduplicator) getLockAttempts(scheduleID int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lockAttempts[scheduleID]
}

// mockAccessKeyInstaller is a test implementation of AccessKeyInstaller
type mockAccessKeyInstaller struct{}

func (m *mockAccessKeyInstaller) Install(key db.AccessKey, usage db.AccessKeyRole, logger task_logger.Logger) (installation ssh.AccessKeyInstallation, err error) {
	return ssh.AccessKeyInstallation{}, nil
}

func setupTestSchedulePool(t *testing.T) (*SchedulePool, db.Store) {
	store := sql.InitConfigCreateTestStore()

	// Store original config and restore after test
	originalSchedule := util.Config.Schedule
	t.Cleanup(func() {
		util.Config.Schedule = originalSchedule
	})

	// Ensure util.Config Schedule is set (CreateTestStore doesn't set this)
	util.Config.Schedule = &util.ScheduleConfig{
		Timezone: "UTC",
	}

	pool := CreateSchedulePool(
		store,
		&tasks.TaskPool{},
		&mockAccessKeyInstaller{},
		&mockEncryptionService{},
	)

	t.Cleanup(func() {
		pool.Destroy()
	})

	return &pool, store
}

// TestSetDeduplicator verifies that SetDeduplicator properly configures the deduplicator
func TestSetDeduplicator(t *testing.T) {
	pool, _ := setupTestSchedulePool(t)

	// Initially no deduplicator should be set
	assert.Nil(t, pool.dedup, "deduplicator should be nil initially")

	// Set a deduplicator
	dedup := newMockDeduplicator()
	pool.SetDeduplicator(dedup)

	// Verify it's set
	assert.NotNil(t, pool.dedup, "deduplicator should be set after calling SetDeduplicator")
	assert.Equal(t, dedup, pool.dedup, "deduplicator should be the one we set")
}

// TestScheduleExecutesNormallyWithoutDeduplicator verifies schedules execute when no deduplicator is set
func TestScheduleExecutesNormallyWithoutDeduplicator(t *testing.T) {
	pool, _ := setupTestSchedulePool(t)

	// Ensure no deduplicator is set
	pool.SetDeduplicator(nil)

	// Verify that the deduplicator is nil (schedule would execute normally)
	assert.Nil(t, pool.dedup, "deduplicator should be nil, allowing normal execution")
}

// TestScheduleSkippedWhenTryLockExecutionReturnsFalse verifies schedules are skipped when TryLockExecution returns false
func TestScheduleSkippedWhenTryLockExecutionReturnsFalse(t *testing.T) {
	pool, _ := setupTestSchedulePool(t)

	// Set up deduplicator to deny execution
	dedup := newMockDeduplicator()
	scheduleID := 123
	dedup.setAllowExecution(scheduleID, false)
	pool.SetDeduplicator(dedup)

	// Simulate the deduplication check that happens in ScheduleRunner.Run()
	shouldSkip := pool.dedup != nil && !pool.dedup.TryLockExecution(scheduleID)

	// Verify the deduplicator was called and returned false
	assert.True(t, shouldSkip, "schedule should be skipped when TryLockExecution returns false")
	assert.Equal(t, 1, dedup.getLockAttempts(scheduleID), "TryLockExecution should be called once")
}

// TestScheduleProceedsWhenTryLockExecutionReturnsTrue verifies schedules proceed when TryLockExecution returns true
func TestScheduleProceedsWhenTryLockExecutionReturnsTrue(t *testing.T) {
	pool, _ := setupTestSchedulePool(t)

	// Set up deduplicator to allow execution
	dedup := newMockDeduplicator()
	scheduleID := 456
	dedup.setAllowExecution(scheduleID, true)
	pool.SetDeduplicator(dedup)

	// Simulate the deduplication check that happens in ScheduleRunner.Run()
	shouldSkip := pool.dedup != nil && !pool.dedup.TryLockExecution(scheduleID)

	// Verify the deduplicator was called and returned true (schedule proceeds)
	assert.False(t, shouldSkip, "schedule should proceed when TryLockExecution returns true")
	assert.Equal(t, 1, dedup.getLockAttempts(scheduleID), "TryLockExecution should be called once")
}

func TestTimezoneChangeReschedulesCronButPreservesRunAt(t *testing.T) {
	pool, store := setupTestSchedulePool(t)
	originalLocal := time.Local
	t.Cleanup(func() { time.Local = originalLocal })
	local, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	time.Local = local
	t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "")
	util.Config.Schedule.Timezone = ""
	project, err := store.CreateProject(db.Project{Name: "schedules"})
	require.NoError(t, err)
	key, err := store.CreateAccessKey(db.AccessKey{ProjectID: &project.ID, Name: "none", Type: db.AccessKeyNone})
	require.NoError(t, err)
	repo, err := store.CreateRepository(db.Repository{ProjectID: project.ID, Name: "repo", GitURL: "/tmp/repo", GitBranch: "main", SSHKeyID: key.ID})
	require.NoError(t, err)
	template, err := store.CreateTemplate(db.Template{ProjectID: project.ID, Name: "scheduled", RepositoryID: repo.ID, App: db.AppBash, Playbook: "test.sh"})
	require.NoError(t, err)
	runAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	for _, schedule := range []db.Schedule{
		{Active: true, CronFormat: "0 9 * * *"},
		{Active: true, CronFormat: "CRON_TZ=Asia/Tokyo 0 9 * * *"},
		{Active: true, Type: db.ScheduleTypeRunAt, RunAt: &runAt},
		{Active: false, CronFormat: "0 12 * * *"},
	} {
		schedule.ProjectID, schedule.TemplateID = project.ID, template.ID
		_, err = store.CreateSchedule(schedule)
		require.NoError(t, err)
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		zone string
		hour int
	}{{"UTC", 9}, {"Europe/Moscow", 6}, {"", 0}} {
		require.NoError(t, store.SetOption(server.ScheduleTimezoneOption, test.zone))
		// Request contexts hold a copy of the pool; both copies must see the refresh.
		copy := *pool
		copy.Refresh()
		entries := pool.cron.Entries()
		require.Len(t, entries, 3)
		for _, entry := range entries {
			next := entry.Schedule.Next(start).UTC()
			switch entry.Job.(ScheduleRunner).scheduleID {
			case 1:
				assert.Equal(t, test.hour, next.Hour())
			case 2:
				assert.Equal(t, 0, next.Hour(), "explicit cron timezone stays intact")
			case 3:
				assert.Equal(t, runAt, next)
			default:
				t.Fatal("unexpected schedule")
			}
		}
		stored, err := store.GetSchedule(project.ID, 3)
		require.NoError(t, err)
		assert.True(t, stored.RunAt.Equal(runAt))
	}
}

func TestSchedulePoolWatchesDatabaseTimezone(t *testing.T) {
	pool, store := setupTestSchedulePool(t)
	t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "")
	util.Config.Schedule.Timezone = ""
	require.NoError(t, store.SetOption(server.ScheduleTimezoneOption, "UTC"))
	pool.Refresh()
	stopped := make(chan struct{})
	go func() { pool.Run(); close(stopped) }()
	t.Cleanup(func() {
		pool.Destroy()
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Error("schedule watcher did not stop")
		}
	})
	// Simulates a settings save on another server: no direct Refresh call.
	require.NoError(t, store.SetOption(server.ScheduleTimezoneOption, "Asia/Tokyo"))
	require.Eventually(t, func() bool {
		pool.locker.Lock()
		defer pool.locker.Unlock()
		return *pool.timezone == "Asia/Tokyo"
	}, 7*time.Second, 20*time.Millisecond)
}
