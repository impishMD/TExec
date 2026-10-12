package server

import (
	"errors"
	"fmt"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/util/appearance"
)

func GetAppearanceTheme(store settingsReader) (string, error) {
	id, err := store.GetOption(appearance.ServerOption)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return "", fmt.Errorf("read appearance theme: %w", err)
	}
	// A rollback to a version with fewer themes must still allow users to log in.
	if !appearance.ValidTheme(id) {
		return appearance.DefaultTheme, nil
	}
	return id, nil
}
