package server

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/impishMD/TExec/db"
)

const RemoteRunnersEnabledOption = "server.remote_runners.enabled"

type ServerSettings struct {
	AppearanceTheme  string                   `json:"appearance_theme"`
	UseRemoteRunner  bool                     `json:"use_remote_runner"`
	ScheduleTimezone ScheduleTimezoneSettings `json:"schedule_timezone"`
}

type settingsReader interface{ GetOption(string) (string, error) }
type settingsWriter interface{ SetOption(string, string) error }

// GetServerSettings reads the database on demand so saves apply to new tasks on
// every server without a restart or a process-local configuration cache.
func GetServerSettings(store settingsReader) (ServerSettings, error) {
	timezone, err := GetScheduleTimezone(store)
	if err != nil {
		return ServerSettings{}, err
	}
	theme, err := GetAppearanceTheme(store)
	if err != nil {
		return ServerSettings{}, err
	}
	settings := ServerSettings{ScheduleTimezone: timezone, AppearanceTheme: theme}
	value, err := store.GetOption(RemoteRunnersEnabledOption)
	if errors.Is(err, db.ErrNotFound) {
		return settings, nil
	}
	if err != nil {
		return ServerSettings{}, fmt.Errorf("read server settings: %w", err)
	}
	switch value {
	case "", "false":
		return settings, nil
	case "true":
		settings.UseRemoteRunner = true
		return settings, nil
	default:
		return ServerSettings{}, errors.New("invalid remote runner setting in the database")
	}
}

func SaveServerSettings(store settingsWriter, settings ServerSettings) error {
	return store.SetOption(RemoteRunnersEnabledOption, strconv.FormatBool(settings.UseRemoteRunner))
}
