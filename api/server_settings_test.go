package api

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/services/audit"
	"github.com/impishMD/TExec/services/server"
	"github.com/impishMD/TExec/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerSettings_AdminOnlyAndDatabaseBacked(t *testing.T) {
	f := newCoreFixture(t)
	util.Config.Mfa.Email = &util.EmailAuthConfig{}
	util.Config.Schedule = &util.ScheduleConfig{}
	util.Config.JWT = &util.JWTConfig{}
	// The settings endpoint must not expose other option values.
	require.NoError(t, f.store.SetOption("unrelated.option", "private-value"))
	for _, actor := range []string{"owner", "manager", "guest", "outsider", ""} {
		status := 403
		if actor == "" {
			status = 401
		}
		for _, method := range []string{"GET", "HEAD", "PUT"} {
			f.request(t, actor, method, "/settings", `{"use_remote_runner":true}`, status)
		}
	}
	w := f.request(t, "admin", "GET", "/settings", "", 200)
	assertRemoteRunnerSetting(t, w, false)
	for _, value := range []string{"true", "false"} {
		body := `{"use_remote_runner":` + value + `}`
		w = f.request(t, "admin", "PUT", "/settings", body, 200)
		assertRemoteRunnerSetting(t, w, value == "true")
		stored, err := f.store.GetOption(server.RemoteRunnersEnabledOption)
		require.NoError(t, err)
		assert.Equal(t, value, stored)
		w = f.request(t, "admin", "GET", "/settings", "", 200)
		assertRemoteRunnerSetting(t, w, value == "true")

		w = f.request(t, "guest", "GET", "/info", "", 200)
		var info struct {
			UseRemoteRunner bool `json:"use_remote_runner"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &info))
		assert.Equal(t, value == "true", info.UseRemoteRunner)
		w = f.request(t, "admin", "GET", "/admin/info", "", 200)
		var adminInfo struct {
			Runners server.ServerSettings `json:"runners"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &adminInfo))
		assert.Equal(t, info.UseRemoteRunner, adminInfo.Runners.UseRemoteRunner)
	}
}

func TestServerSettings_InvalidUpdatesPreserveSavedValue(t *testing.T) {
	f := newCoreFixture(t)
	f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":true}`, 200)
	for _, body := range []string{"", "{", "{}", "null", `{"use_remote_runner":null}`,
		`{"use_remote_runner":"false"}`, `{"use_remote_runner":0}`} {
		f.request(t, "admin", "PUT", "/settings", body, 400)
	}
	f.request(t, "admin", "POST", "/options", `{"key":"`+server.RemoteRunnersEnabledOption+`","value":"invalid"}`, 400)
	w := f.request(t, "admin", "GET", "/settings", "", 200)
	assertRemoteRunnerSetting(t, w, true)
	// A malformed value must not be reported as a successful disabled state.
	require.NoError(t, f.store.SetOption(server.RemoteRunnersEnabledOption, "invalid"))
	f.request(t, "admin", "GET", "/settings", "", 500)
}

type failingServerSettingsStore struct{ db.Store }

func (failingServerSettingsStore) SetOption(string, string) error { return errors.New("write failed") }

func TestServerSettings_AuditOnlySuccessfulSave(t *testing.T) {
	store := setupSessionTest(t)
	admin := makeAdmin(t, store, createUserOptionsTestUser(t, store, "admin"))
	r, rec := userRequest(store, "PUT", "/api/settings", `{"use_remote_runner":true}`, admin, db.User{})
	w := httptest.NewRecorder()
	serverSettings(w, r)
	require.Equal(t, 200, w.Code)
	assert.Equal(t, audit.SettingsMetadata{Keys: []string{server.RemoteRunnersEnabledOption}},
		onlyEvent(t, rec, audit.SystemSettingsUpdate).Event.Metadata)

	r, rec = userRequest(failingServerSettingsStore{store}, "PUT", "/api/settings",
		`{"use_remote_runner":false}`, admin, db.User{})
	w = httptest.NewRecorder()
	serverSettings(w, r)
	require.Equal(t, 500, w.Code)
	assert.Empty(t, rec.All())
	settings, err := server.GetServerSettings(store)
	require.NoError(t, err)
	assert.True(t, settings.UseRemoteRunner)
}

func assertRemoteRunnerSetting(t *testing.T, w *httptest.ResponseRecorder, expected bool) {
	t.Helper()
	var settings server.ServerSettings
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &settings))
	assert.Equal(t, expected, settings.UseRemoteRunner)
	assert.NotEmpty(t, settings.ScheduleTimezone.Value)
	assert.NotContains(t, w.Body.String(), "private-value")
}

func TestServerSettings_ScheduleTimezone(t *testing.T) {
	f := newCoreFixture(t)
	t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "")
	util.Config.Schedule = &util.ScheduleConfig{}
	util.Config.Mfa.Email = &util.EmailAuthConfig{}
	util.Config.JWT = &util.JWTConfig{}
	read := func() server.ScheduleTimezoneSettings {
		w := f.request(t, "admin", "GET", "/settings", "", 200)
		var settings server.ServerSettings
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &settings))
		return settings.ScheduleTimezone
	}
	assert.Equal(t, "system", read().Source)
	for _, value := range []string{"Europe/Moscow", "America/New_York", ""} {
		body, err := json.Marshal(map[string]any{"use_remote_runner": false, "schedule_timezone": value})
		require.NoError(t, err)
		f.request(t, "admin", "PUT", "/settings", string(body), 200)
		stored, err := f.store.GetOption(server.ScheduleTimezoneOption)
		require.NoError(t, err)
		assert.Equal(t, value, stored)
		timezone := read()
		assert.Equal(t, value, timezone.Saved)
		if value == "" {
			assert.Equal(t, "system", timezone.Source)
		} else {
			assert.Equal(t, value, timezone.Value)
			assert.Equal(t, "database", timezone.Source)
		}
		w := f.request(t, "guest", "GET", "/info", "", 200)
		var info SystemInfo
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &info))
		assert.Equal(t, timezone.Value, info.ScheduleTimezone)
	}
	f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":false,"schedule_timezone":"Europe/Moscow"}`, 200)
	for _, value := range []string{"Invalid/Zone", "Local", " Europe/Moscow "} {
		body, err := json.Marshal(map[string]any{"use_remote_runner": true, "schedule_timezone": value})
		require.NoError(t, err)
		f.request(t, "admin", "PUT", "/settings", string(body), 400)
		stored, err := f.store.GetOption(server.RemoteRunnersEnabledOption)
		require.NoError(t, err)
		assert.Equal(t, "false", stored, "validation must precede all writes")
		assert.Equal(t, "Europe/Moscow", read().Saved)
	}
	for _, source := range []string{"configuration", "environment"} {
		util.Config.Schedule.Timezone = "Asia/Tokyo"
		if source == "environment" {
			t.Setenv("TEXEC_SCHEDULE_TIMEZONE", "Europe/Paris")
		}
		timezone := read()
		assert.Equal(t, source, timezone.Source)
		assert.True(t, timezone.Locked())
		assert.Equal(t, "Europe/Moscow", timezone.Saved)
		f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":true,"schedule_timezone":"UTC"}`, 400)
		f.request(t, "admin", "POST", "/options", `{"key":"schedule.timezone","value":"UTC"}`, 400)
		// Other settings can still be changed without touching the locked value.
		f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":true}`, 200)
		assert.Equal(t, timezone, read())
	}
}
