package linux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/scheduler"
)

const cronFileName = "teams-quiet-hours"

type SchedulerManager struct{}

func cronPath() string {
	if override := os.Getenv("TEAMS_QUIET_HOURS_CRON_FILE"); override != "" {
		return override
	}
	return linuxSystemPath("etc", "cron.d", cronFileName)
}

func systemdAvailable() bool {
	if os.Getenv("TEAMS_QUIET_HOURS_SCHEDULER") == "cron" {
		return false
	}
	if os.Getenv("TEAMS_QUIET_HOURS_SCHEDULER") == "systemd" {
		return true
	}
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func systemdDir() string {
	if override := os.Getenv("TEAMS_QUIET_HOURS_SYSTEMD_DIR"); override != "" {
		return override
	}
	return linuxSystemPath("etc", "systemd", "system")
}

func cronField(value string) (hour string, minute string, err error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid time %q", value)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", "", err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", "", err
	}
	return strconv.Itoa(h), strconv.Itoa(m), nil
}

func (SchedulerManager) Install(cfg config.Config, executable string, dryRun bool) error {
	if systemdAvailable() {
		return installSystemd(cfg, executable, dryRun)
	}
	startHour, startMin, err := cronField(cfg.Start)
	if err != nil {
		return err
	}
	endHour, endMin, err := cronField(cfg.End)
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("SHELL=/bin/sh\n")
	b.WriteString("PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\n")
	if cfg.Timezone != "" {
		fmt.Fprintf(&b, "CRON_TZ=%s\n", cfg.Timezone)
	}
	fmt.Fprintf(&b, "%s %s * * 1-5 root %s allow --browser %s\n", startMin, startHour, executable, cfg.Browser)
	fmt.Fprintf(&b, "%s %s * * 1-5 root %s block --browser %s\n", endMin, endHour, executable, cfg.Browser)
	fmt.Fprintf(&b, "0 0 * * 6,0 root %s block --browser %s\n", executable, cfg.Browser)
	if cfg.Mode == "nuclear" {
		fmt.Fprintf(&b, "*/%d * * * * root %s enforce\n", cfg.EnforceEveryMins, executable)
	}

	path := cronPath()
	if dryRun {
		fmt.Printf("DRY-RUN: write %s\n%s", path, b.String())
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func installSystemd(cfg config.Config, executable string, dryRun bool) error {
	jobs := map[string]string{
		"teams-quiet-hours-allow.service": fmt.Sprintf("[Unit]\nDescription=Allow Teams quiet-hours\n\n[Service]\nType=oneshot\nExecStart=%s allow --browser %s\n", executable, cfg.Browser),
		"teams-quiet-hours-block.service": fmt.Sprintf("[Unit]\nDescription=Block Teams quiet-hours\n\n[Service]\nType=oneshot\nExecStart=%s block --browser %s\n", executable, cfg.Browser),
	}
	timers := map[string]string{
		"teams-quiet-hours-allow.timer": fmt.Sprintf("[Unit]\nDescription=Allow Teams on workdays\n\n[Timer]\nOnCalendar=Mon..Fri %s:00\nPersistent=true\n\n[Install]\nWantedBy=timers.target\n", cfg.Start),
		"teams-quiet-hours-block.timer": fmt.Sprintf("[Unit]\nDescription=Block Teams outside work hours\n\n[Timer]\nOnCalendar=Mon..Fri %s:00\nOnCalendar=Sat,Sun 00:00:00\nPersistent=true\n\n[Install]\nWantedBy=timers.target\n", cfg.End),
	}
	if cfg.Mode == "nuclear" {
		jobs["teams-quiet-hours-enforce.service"] = fmt.Sprintf("[Unit]\nDescription=Enforce Teams quiet-hours\n\n[Service]\nType=oneshot\nExecStart=%s enforce\n", executable)
		timers["teams-quiet-hours-enforce.timer"] = fmt.Sprintf("[Unit]\nDescription=Repeated Teams quiet-hours enforcement\n\n[Timer]\nOnBootSec=1min\nOnUnitActiveSec=%dmin\nPersistent=true\n\n[Install]\nWantedBy=timers.target\n", cfg.EnforceEveryMins)
	}
	for name, content := range jobs {
		if err := writeSystemdFile(name, content, dryRun); err != nil {
			return err
		}
	}
	for name, content := range timers {
		if err := writeSystemdFile(name, content, dryRun); err != nil {
			return err
		}
		if !dryRun {
			_ = execSystemctl("enable", "--now", name)
		}
	}
	return nil
}

func writeSystemdFile(name string, content string, dryRun bool) error {
	path := filepath.Join(systemdDir(), name)
	if dryRun {
		fmt.Printf("DRY-RUN: write %s\n%s", path, content)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func execSystemctl(args ...string) error {
	// Best effort: some distros/containers do not allow systemctl even with systemd paths.
	_ = exec.Command("systemctl", append([]string{"daemon-reload"}, []string{}...)...).Run()
	return exec.Command("systemctl", args...).Run()
}

func (SchedulerManager) Uninstall(dryRun bool) error {
	if systemdAvailable() {
		for _, name := range []string{
			"teams-quiet-hours-allow.timer", "teams-quiet-hours-block.timer", "teams-quiet-hours-enforce.timer",
			"teams-quiet-hours-allow.service", "teams-quiet-hours-block.service", "teams-quiet-hours-enforce.service",
		} {
			path := filepath.Join(systemdDir(), name)
			if dryRun {
				fmt.Printf("DRY-RUN: remove %s\n", path)
				continue
			}
			err := os.Remove(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	path := cronPath()
	if dryRun {
		fmt.Printf("DRY-RUN: remove %s\n", path)
		return nil
	}
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (SchedulerManager) Status() ([]scheduler.Status, error) {
	if systemdAvailable() {
		var statuses []scheduler.Status
		for _, name := range []string{"teams-quiet-hours-allow.timer", "teams-quiet-hours-block.timer", "teams-quiet-hours-enforce.timer"} {
			path := filepath.Join(systemdDir(), name)
			_, err := os.Stat(path)
			installed := err == nil
			detail := "systemd timer present"
			if errors.Is(err, os.ErrNotExist) {
				detail = "systemd timer absent"
			} else if err != nil {
				detail = err.Error()
			}
			statuses = append(statuses, scheduler.Status{Name: name, Target: path, Installed: installed, Detail: detail})
		}
		return statuses, nil
	}
	path := cronPath()
	_, err := os.Stat(path)
	installed := err == nil
	detail := "cron file present"
	if errors.Is(err, os.ErrNotExist) {
		detail = "cron file absent"
	} else if err != nil {
		detail = err.Error()
	}
	return []scheduler.Status{{
		Name:      "cron",
		Target:    path,
		Installed: installed,
		Detail:    detail,
	}}, nil
}
