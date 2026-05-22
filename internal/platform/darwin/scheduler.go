package darwin

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/scheduler"
)

type SchedulerManager struct{}

type launchJob struct {
	Name     string
	Command  string
	Time     string
	Weekdays []int
}

func launchDaemonsDir() string {
	return filepath.Join(macPrefix(), "Library", "LaunchDaemons")
}

func launchPath(name string) string {
	return filepath.Join(launchDaemonsDir(), name+".plist")
}

func parseTime(value string) (int, int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time %q", value)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return hour, minute, nil
}

func launchPlist(label string, executable string, args []string, hour int, minute int, weekdays []int) string {
	var cal strings.Builder
	if len(weekdays) == 1 {
		fmt.Fprintf(&cal, "<dict><key>Hour</key><integer>%d</integer><key>Minute</key><integer>%d</integer><key>Weekday</key><integer>%d</integer></dict>", hour, minute, weekdays[0])
	} else {
		cal.WriteString("<array>")
		for _, weekday := range weekdays {
			fmt.Fprintf(&cal, "<dict><key>Hour</key><integer>%d</integer><key>Minute</key><integer>%d</integer><key>Weekday</key><integer>%d</integer></dict>", hour, minute, weekday)
		}
		cal.WriteString("</array>")
	}
	var argv strings.Builder
	fmt.Fprintf(&argv, "<string>%s</string>", xmlEscape(executable))
	for _, arg := range args {
		fmt.Fprintf(&argv, "<string>%s</string>", xmlEscape(arg))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key><array>%s</array>
  <key>StartCalendarInterval</key>%s
</dict>
</plist>
`, xmlEscape(label), argv.String(), cal.String())
}

func xmlEscape(value string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(value)); err != nil {
		return value
	}
	return b.String()
}

func (SchedulerManager) Install(cfg config.Config, executable string, dryRun bool) error {
	startHour, startMinute, err := parseTime(cfg.Start)
	if err != nil {
		return err
	}
	endHour, endMinute, err := parseTime(cfg.End)
	if err != nil {
		return err
	}
	jobs := []struct {
		name     string
		command  string
		hour     int
		minute   int
		weekdays []int
	}{
		{"com.teams-quiet-hours.allow-weekdays", "allow", startHour, startMinute, []int{1, 2, 3, 4, 5}},
		{"com.teams-quiet-hours.block-weekdays", "block", endHour, endMinute, []int{1, 2, 3, 4, 5}},
		{"com.teams-quiet-hours.block-weekends", "block", 0, 0, []int{0, 6}},
	}
	for _, job := range jobs {
		path := launchPath(job.name)
		content := launchPlist(job.name, executable, []string{job.command, "--browser", cfg.Browser}, job.hour, job.minute, job.weekdays)
		if dryRun {
			fmt.Printf("DRY-RUN: write %s\n%s", path, content)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	if cfg.Mode == "nuclear" {
		path := launchPath("com.teams-quiet-hours.enforce")
		content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.teams-quiet-hours.enforce</string>
  <key>ProgramArguments</key><array><string>%s</string><string>enforce</string></array>
  <key>StartInterval</key><integer>%d</integer>
</dict>
</plist>
`, xmlEscape(executable), cfg.EnforceEveryMins*60)
		if dryRun {
			fmt.Printf("DRY-RUN: write %s\n%s", path, content)
		} else {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func (SchedulerManager) Uninstall(dryRun bool) error {
	for _, name := range []string{"com.teams-quiet-hours.allow-weekdays", "com.teams-quiet-hours.block-weekdays", "com.teams-quiet-hours.block-weekends", "com.teams-quiet-hours.enforce"} {
		path := launchPath(name)
		if dryRun {
			fmt.Printf("DRY-RUN: remove %s\n", path)
			continue
		}
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (SchedulerManager) Status() ([]scheduler.Status, error) {
	var statuses []scheduler.Status
	for _, name := range []string{"com.teams-quiet-hours.allow-weekdays", "com.teams-quiet-hours.block-weekdays", "com.teams-quiet-hours.block-weekends", "com.teams-quiet-hours.enforce"} {
		path := launchPath(name)
		_, err := os.Stat(path)
		installed := err == nil
		detail := "LaunchDaemon present"
		if errors.Is(err, os.ErrNotExist) {
			detail = "LaunchDaemon absent"
		} else if err != nil {
			detail = err.Error()
		}
		statuses = append(statuses, scheduler.Status{Name: name, Target: path, Installed: installed, Detail: detail})
	}
	return statuses, nil
}
