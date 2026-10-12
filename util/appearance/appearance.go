// Package appearance defines the shared catalog of built-in service themes.
package appearance

import (
	_ "embed"
	"encoding/json"
)

const DefaultTheme = "texec"
const ServerOption = "server.appearance.theme"
const UserOption = "appearance.theme"

// The WebUI imports this same catalog; stored preferences contain only an ID.
//
//go:embed themes.json
var catalog []byte

var themes = func() map[string]bool {
	var entries []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(catalog, &entries); err != nil {
		panic(err)
	}
	ids := make(map[string]bool, len(entries))
	for _, entry := range entries {
		ids[entry.ID] = true
	}
	return ids
}()

func ValidTheme(id string) bool { return themes[id] }
