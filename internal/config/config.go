package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	AppName      = "teams-quiet-hours"
	DefaultStart = "09:00"
	DefaultEnd   = "17:00"
	DefaultMode  = "soft"
)

var timePattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

type Config struct {
	Mode             string `json:"mode"`
	Browser          string `json:"browser"`
	Start            string `json:"start"`
	End              string `json:"end"`
	Timezone         string `json:"timezone,omitempty"`
	Weekdays         string `json:"weekdays"`
	LaunchOnAllow    bool   `json:"launch_on_allow"`
	CloseBrowserTabs bool   `json:"close_browser_tabs"`
	BlockNetwork     bool   `json:"block_network"`
	EnforceEveryMins int    `json:"enforce_every_mins"`
	Paused           bool   `json:"paused"`
}

type Options struct {
	Mode             string
	Browser          string
	Start            string
	End              string
	Timezone         string
	Weekdays         string
	LaunchOnAllow    bool
	LaunchOnAllowSet bool
	CloseBrowserTabs bool
	CloseTabsSet     bool
	BlockNetwork     bool
	BlockNetworkSet  bool
	Paused           bool
	PausedSet        bool
	DryRun           bool
}

func Default() Config {
	return Config{
		Mode:             DefaultMode,
		Browser:          "chrome",
		Start:            DefaultStart,
		End:              DefaultEnd,
		Weekdays:         "mon-fri",
		EnforceEveryMins: 5,
	}
}

func Merge(base Config, opts Options) Config {
	if opts.Mode != "" {
		base.Mode = opts.Mode
	}
	if opts.Browser != "" {
		base.Browser = opts.Browser
	}
	if opts.Start != "" {
		base.Start = opts.Start
	}
	if opts.End != "" {
		base.End = opts.End
	}
	if opts.Timezone != "" {
		base.Timezone = opts.Timezone
	}
	if opts.Weekdays != "" {
		base.Weekdays = opts.Weekdays
	}
	if opts.LaunchOnAllowSet {
		base.LaunchOnAllow = opts.LaunchOnAllow
	}
	if opts.CloseTabsSet {
		base.CloseBrowserTabs = opts.CloseBrowserTabs
	}
	if opts.BlockNetworkSet {
		base.BlockNetwork = opts.BlockNetwork
	}
	if opts.PausedSet {
		base.Paused = opts.Paused
	}
	return base
}

func Validate(cfg Config) error {
	switch cfg.Mode {
	case "soft", "hard", "nuclear":
	default:
		return fmt.Errorf("mode must be soft, hard, or nuclear")
	}
	switch cfg.Browser {
	case "chrome", "chromium", "edge", "all":
	default:
		return fmt.Errorf("browser must be chrome, chromium, edge, or all")
	}
	if !timePattern.MatchString(cfg.Start) {
		return fmt.Errorf("start must use HH:MM in 24-hour time")
	}
	if !timePattern.MatchString(cfg.End) {
		return fmt.Errorf("end must use HH:MM in 24-hour time")
	}
	if _, err := WeekdaySet(cfg.Weekdays); err != nil {
		return err
	}
	if cfg.EnforceEveryMins <= 0 {
		return fmt.Errorf("enforce interval must be greater than zero")
	}
	return nil
}

func WeekdaySet(value string) (map[time.Weekday]bool, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "mon-fri" || value == "weekdays" {
		return map[time.Weekday]bool{
			time.Monday: true, time.Tuesday: true, time.Wednesday: true, time.Thursday: true, time.Friday: true,
		}, nil
	}
	lookup := map[string]time.Weekday{
		"sun": time.Sunday, "sunday": time.Sunday,
		"mon": time.Monday, "monday": time.Monday,
		"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
		"wed": time.Wednesday, "wednesday": time.Wednesday,
		"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
		"fri": time.Friday, "friday": time.Friday,
		"sat": time.Saturday, "saturday": time.Saturday,
	}
	out := map[time.Weekday]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		day, ok := lookup[part]
		if !ok {
			return nil, fmt.Errorf("unsupported weekday %q", part)
		}
		out[day] = true
	}
	return out, nil
}

func IsAllowedNow(cfg Config, now time.Time) (bool, error) {
	if cfg.Timezone != "" {
		loc, err := time.LoadLocation(cfg.Timezone)
		if err != nil {
			return false, err
		}
		now = now.In(loc)
	}
	weekdays, err := WeekdaySet(cfg.Weekdays)
	if err != nil {
		return false, err
	}
	if !weekdays[now.Weekday()] {
		return false, nil
	}
	start, err := parseClock(cfg.Start, now)
	if err != nil {
		return false, err
	}
	end, err := parseClock(cfg.End, now)
	if err != nil {
		return false, err
	}
	return !now.Before(start) && now.Before(end), nil
}

func parseClock(value string, base time.Time) (time.Time, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time %q", value)
	}
	var hour, minute int
	if _, err := fmt.Sscanf(value, "%02d:%02d", &hour, &minute); err != nil {
		return time.Time{}, err
	}
	return time.Date(base.Year(), base.Month(), base.Day(), hour, minute, 0, 0, base.Location()), nil
}

func Dir() string {
	if override := os.Getenv("TEAMS_QUIET_HOURS_CONFIG_DIR"); override != "" {
		return override
	}
	if prefix := os.Getenv("TEAMS_QUIET_HOURS_SYSTEM_PREFIX"); prefix != "" {
		return filepath.Join(prefix, "etc", AppName)
	}

	switch runtime.GOOS {
	case "darwin":
		return filepath.Join("/Library", "Application Support", AppName)
	case "windows":
		if programData := os.Getenv("ProgramData"); programData != "" {
			return filepath.Join(programData, AppName)
		}
		return filepath.Join(`C:\ProgramData`, AppName)
	default:
		return filepath.Join("/etc", AppName)
	}
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load() (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, Validate(cfg)
}

func Save(cfg Config, dryRun bool) error {
	if err := Validate(cfg); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if dryRun {
		fmt.Printf("DRY-RUN: write %s\n%s", Path(), string(data))
		return nil
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o644)
}

func Remove(dryRun bool) error {
	if dryRun {
		fmt.Printf("DRY-RUN: remove %s\n", Path())
		return nil
	}
	err := os.Remove(Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
