package server

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/impishMD/TExec/db"
	"github.com/impishMD/TExec/util"
)

const ScheduleTimezoneOption = "schedule.timezone"

var ErrScheduleTimezoneLocked = errors.New("schedule timezone is managed by environment or configuration")
var ErrScheduleTimezoneInvalid = errors.New("invalid schedule timezone")

type ScheduleTimezoneSettings struct {
	Value    string         `json:"value"`
	Saved    string         `json:"saved"`
	Source   string         `json:"source"`
	System   string         `json:"system"`
	Location *time.Location `json:"-"`
}

func (s ScheduleTimezoneSettings) Locked() bool {
	return s.Source == "environment" || s.Source == "configuration"
}

// systemTimezoneName exposes an IANA name for browser previews. Go may call
// the same location "Local" when /etc/localtime supplies the system zone.
func systemTimezoneName() string {
	if name := time.Local.String(); name != "Local" {
		return name
	}
	return detectSystemTimezoneName()
}

var detectSystemTimezoneName = sync.OnceValue(func() string {
	candidates := []string{strings.TrimPrefix(os.Getenv("TZ"), ":")}
	if path, err := filepath.EvalSymlinks("/etc/localtime"); err == nil {
		if _, name, ok := strings.Cut(filepath.ToSlash(path), "/zoneinfo/"); ok {
			candidates = append(candidates, name)
		}
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		candidates = append(candidates, strings.TrimSpace(string(data)))
	}
	for _, name := range candidates {
		if name != "" && name != "Local" {
			if _, err := time.LoadLocation(name); err == nil {
				return name
			}
		}
	}
	// Some distributions copy the zone file instead of using a symlink.
	if local, err := os.ReadFile("/etc/localtime"); err == nil {
		for _, root := range []string{"/usr/share/zoneinfo", "/usr/share/lib/zoneinfo"} {
			if name := matchSystemZoneFile(root, local); name != "" {
				return name
			}
		}
	}
	// UTC systems (including the default container) need no named zone file.
	name, offset := time.Now().In(time.Local).Zone()
	if offset == 0 && (name == "UTC" || name == "GMT") {
		return "UTC"
	}
	return time.Local.String()
})

func matchSystemZoneFile(root string, local []byte) string {
	table, err := os.ReadFile(filepath.Join(root, "zone.tab"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(table), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || strings.HasPrefix(line, "#") {
			continue
		}
		if data, err := os.ReadFile(filepath.Join(root, fields[2])); err == nil && bytes.Equal(data, local) {
			return fields[2]
		}
	}
	return ""
}

func GetScheduleTimezone(store settingsReader) (ScheduleTimezoneSettings, error) {
	saved, err := store.GetOption(ScheduleTimezoneOption)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return ScheduleTimezoneSettings{}, fmt.Errorf("read schedule timezone: %w", err)
	}
	setting := ScheduleTimezoneSettings{Saved: saved, System: systemTimezoneName(), Source: "system", Location: time.Local}
	setting.Value = setting.System
	value := ""
	if env := os.Getenv("TEXEC_SCHEDULE_TIMEZONE"); env != "" {
		value, setting.Source = env, "environment"
	} else if util.Config.Schedule != nil && util.Config.Schedule.Timezone != "" {
		value, setting.Source = util.Config.Schedule.Timezone, "configuration"
	} else if saved != "" {
		value, setting.Source = saved, "database"
	}
	if value != "" {
		setting.Location, err = time.LoadLocation(value)
		if err != nil {
			return ScheduleTimezoneSettings{}, fmt.Errorf("resolve schedule timezone: %w", err)
		}
		setting.Value = value
		if value == "Local" {
			setting.Value = setting.System
		}
	}
	return setting, nil
}

func ValidateScheduleTimezone(value string, current ScheduleTimezoneSettings) error {
	if current.Locked() {
		return ErrScheduleTimezoneLocked
	}
	if value == "" {
		return nil // Follow the server's system timezone.
	}
	if value == "Local" || strings.TrimSpace(value) != value {
		return ErrScheduleTimezoneInvalid
	}
	if _, err := time.LoadLocation(value); err != nil {
		return ErrScheduleTimezoneInvalid
	}
	return nil
}
