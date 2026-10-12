package api

import (
	"net/http"

	"github.com/impishMD/TExec/api/helpers"
	"github.com/impishMD/TExec/services/server"
)

// Public branding for the login page. Never expose the full server settings.
func getAppearance(w http.ResponseWriter, r *http.Request) {
	theme, err := server.GetAppearanceTheme(helpers.Store(r))
	if err != nil {
		writeServerSettingsError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"theme": theme})
}
