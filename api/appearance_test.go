package api

import (
	"encoding/json"
	"testing"

	"github.com/impishMD/TExec/util/appearance"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppearancePublicDefaultAndAdminUpdate(t *testing.T) {
	f := newCoreFixture(t)
	read := func() string {
		w := f.request(t, "", "GET", "/appearance", "", 200)
		var result map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
		require.Len(t, result, 1, "public appearance must not expose other settings")
		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		return result["theme"]
	}
	assert.Equal(t, "texec", read())
	for _, actor := range []string{"guest", "owner", "manager"} {
		f.request(t, actor, "PUT", "/settings", `{"use_remote_runner":false,"appearance_theme":"nord"}`, 403)
	}
	f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":false,"appearance_theme":"nord"}`, 200)
	assert.Equal(t, "nord", read())
	stored, err := f.store.GetOption(appearance.ServerOption)
	require.NoError(t, err)
	assert.Equal(t, "nord", stored)
	// Old clients updating unrelated settings must preserve the chosen theme.
	f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":true}`, 200)
	assert.Equal(t, "nord", read())
	for _, id := range []string{"", "unknown", "#ffffff", "<script>"} {
		data, err := json.Marshal(map[string]any{"use_remote_runner": false, "appearance_theme": id})
		require.NoError(t, err)
		f.request(t, "admin", "PUT", "/settings", string(data), 400)
		data, err = json.Marshal(map[string]string{"key": appearance.ServerOption, "value": id})
		require.NoError(t, err)
		f.request(t, "admin", "POST", "/options", string(data), 400)
		assert.Equal(t, "nord", read())
		assertRemoteRunnerSetting(t, f.request(t, "admin", "GET", "/settings", "", 200), true)
	}
	f.request(t, "admin", "PUT", "/settings", `{"use_remote_runner":false,"appearance_theme":"texec"}`, 200)
	assert.Equal(t, "texec", read())
	require.NoError(t, f.store.SetOption(appearance.ServerOption, "theme-from-newer-version"))
	assert.Equal(t, "texec", read(), "unknown stored themes should not break the login page")
}

func TestAppearanceUserPreferenceValidationAndIsolation(t *testing.T) {
	f := newCoreFixture(t)
	for _, id := range []string{"ocean", "contrast", "amber", "blue-orange", ""} {
		value, err := json.Marshal(id)
		require.NoError(t, err)
		body, err := json.Marshal(map[string]string{"key": appearance.UserOption, "value": string(value)})
		require.NoError(t, err)
		f.request(t, "guest", "POST", "/user/options", string(body), 200)
		w := f.request(t, "guest", "GET", "/user/options", "", 200)
		var options map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &options))
		assert.Equal(t, string(value), options[appearance.UserOption])
		w = f.request(t, "manager", "GET", "/user/options", "", 200)
		var other map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &other))
		assert.NotContains(t, other, appearance.UserOption)
	}
	for _, value := range []string{`null`, `true`, `42`, `{}`, `[]`, `"unknown"`, `"#ffffff"`} {
		body, err := json.Marshal(map[string]string{"key": appearance.UserOption, "value": value})
		require.NoError(t, err)
		f.request(t, "guest", "POST", "/user/options", string(body), 400)
	}
}
