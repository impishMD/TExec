package api

import (
	"net/http"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/services/audit"
	"github.com/impishMD/TExec/services/schedules"
	"github.com/impishMD/TExec/services/server"
	"github.com/impishMD/TExec/util/appearance"
	log "github.com/sirupsen/logrus"
)

func serverSettings(w http.ResponseWriter, r *http.Request) {
	if !helpers.GetFromContext(r, "user").(*db.User).Admin {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	store := helpers.Store(r)
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		settings, err := server.GetServerSettings(store)
		if err != nil {
			writeServerSettingsError(w, err)
			return
		}
		helpers.WriteJSON(w, http.StatusOK, settings)
		return
	}
	var update struct {
		AppearanceTheme  *string `json:"appearance_theme"`
		UseRemoteRunner  *bool   `json:"use_remote_runner"`
		ScheduleTimezone *string `json:"schedule_timezone"`
	}
	if !helpers.Bind(w, r, &update) {
		return
	}
	if update.UseRemoteRunner == nil {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "use_remote_runner must be a boolean"})
		return
	}
	settings, err := server.GetServerSettings(store)
	if err != nil {
		writeServerSettingsError(w, err)
		return
	}
	if update.ScheduleTimezone != nil {
		if err = server.ValidateScheduleTimezone(*update.ScheduleTimezone, settings.ScheduleTimezone); err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	if update.AppearanceTheme != nil && !appearance.ValidTheme(*update.AppearanceTheme) {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown appearance theme"})
		return
	}
	settings.UseRemoteRunner = *update.UseRemoteRunner
	if err := server.SaveServerSettings(store, settings); err != nil {
		writeServerSettingsError(w, err)
		return
	}
	keys := []string{server.RemoteRunnersEnabledOption}
	if update.ScheduleTimezone != nil {
		if err = store.SetOption(server.ScheduleTimezoneOption, *update.ScheduleTimezone); err != nil {
			writeServerSettingsError(w, err)
			return
		}
		keys = append(keys, server.ScheduleTimezoneOption)
		refreshServerSchedules(r)
	}
	if update.AppearanceTheme != nil {
		if err = store.SetOption(appearance.ServerOption, *update.AppearanceTheme); err != nil {
			writeServerSettingsError(w, err)
			return
		}
		keys = append(keys, appearance.ServerOption)
	}
	helpers.Audit(r).Record(r.Context(), audit.Event{
		Kind:     audit.SystemSettingsUpdate,
		Metadata: audit.SettingsMetadata{Keys: keys},
	})
	settings, err = server.GetServerSettings(store)
	if err != nil {
		writeServerSettingsError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, settings)
}

func refreshServerSchedules(r *http.Request) {
	if pool, ok := helpers.GetFromContext(r, "schedule_pool").(schedules.SchedulePool); ok {
		pool.Refresh()
	}
}

func writeServerSettingsError(w http.ResponseWriter, err error) {
	log.WithError(err).Error("Server settings operation failed")
	w.WriteHeader(http.StatusInternalServerError)
}
