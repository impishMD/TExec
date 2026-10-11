package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/db/sql"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemTimezoneCopiedZoneFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "Europe"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "zone.tab"), []byte("# comment\nRU\t+5545+03735\tEurope/Moscow\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "Europe/Moscow"), []byte("zone data"), 0600))
	assert.Equal(t, "Europe/Moscow", matchSystemZoneFile(root, []byte("zone data")))
	assert.Empty(t, matchSystemZoneFile(root, []byte("other zone")))
}

func TestScheduleTimezonePrecedenceAndRestart(t *testing.T) {
	oldConfig, oldLocal := util.Config, time.Local
	t.Cleanup(func() { util.Config, time.Local = oldConfig, oldLocal })
	store := sql.InitConfigCreateTestStore()
	defer store.Close()
	t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "")
	var err error
	time.Local, err = time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	util.Config.Schedule = &util.ScheduleConfig{}
	check := func(value, source string) ScheduleTimezoneSettings {
		t.Helper()
		got, err := GetScheduleTimezone(store)
		require.NoError(t, err)
		assert.Equal(t, value, got.Value)
		assert.Equal(t, source, got.Source)
		assert.Equal(t, "Asia/Tokyo", got.System)
		return got
	}
	assert.Same(t, time.Local, check("Asia/Tokyo", "system").Location)
	require.NoError(t, store.SetOption(ScheduleTimezoneOption, "America/New_York"))
	check("America/New_York", "database")
	// Startup loading must never promote a DB preference into an external override.
	require.NoError(t, db.FillConfigFromDB(store))
	assert.Empty(t, util.Config.Schedule.Timezone)
	check("America/New_York", "database")
	util.Config.Schedule.Timezone = "Europe/Paris"
	require.NoError(t, db.FillConfigFromDB(store))
	assert.Equal(t, "Europe/Paris", util.Config.Schedule.Timezone)
	check("Europe/Paris", "configuration")
	t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "Europe/Moscow")
	setting := check("Europe/Moscow", "environment")
	assert.Equal(t, "America/New_York", setting.Saved)
	assert.ErrorIs(t, ValidateScheduleTimezone("UTC", setting), ErrScheduleTimezoneLocked)
	t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "")
	util.Config.Schedule.Timezone = ""
	check("America/New_York", "database")
	require.NoError(t, store.SetOption(ScheduleTimezoneOption, ""))
	check("Asia/Tokyo", "system")
}
