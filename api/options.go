package api

import (
	"net/http"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/services/audit"
	"github.com/impishMD/TExec/services/server"
)

func setOption(w http.ResponseWriter, r *http.Request) {
	currentUser := helpers.GetFromContext(r, "user").(*db.User)

	if !currentUser.Admin {
		helpers.WriteJSON(w, http.StatusForbidden, map[string]string{
			"error": "User must be admin",
		})
		return
	}

	var option db.Option
	if !helpers.Bind(w, r, &option) {
		return
	}

	if option.Key == server.RemoteRunnersEnabledOption && option.Value != "true" && option.Value != "false" {
		helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "remote runner setting must be true or false"})
		return
	}
	if option.Key == server.ScheduleTimezoneOption {
		current, err := server.GetScheduleTimezone(helpers.Store(r))
		if err != nil {
			writeServerSettingsError(w, err)
			return
		}
		if err = server.ValidateScheduleTimezone(option.Value, current); err != nil {
			helpers.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	err := helpers.Store(r).SetOption(option.Key, option.Value)
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Can not set option",
		})
		return
	}
	if option.Key == server.ScheduleTimezoneOption {
		refreshServerSchedules(r)
	}

	helpers.Audit(r).Record(r.Context(), audit.Event{
		Kind:     audit.SystemSettingsUpdate,
		Metadata: audit.SettingsMetadata{Keys: []string{audit.TruncateName(option.Key, audit.MaxNameBytes)}},
	})

	helpers.WriteJSON(w, http.StatusOK, option)
}

func getOptions(w http.ResponseWriter, r *http.Request) {
	currentUser := helpers.GetFromContext(r, "user").(*db.User)

	if !currentUser.Admin {
		helpers.WriteJSON(w, http.StatusForbidden, map[string]string{
			"error": "User must be admin",
		})
		return
	}

	options, err := helpers.Store(r).GetOptions(db.RetrieveQueryParams{})
	if err != nil {
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Can not get options",
		})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, options)
}
