package windows

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/scheduler"
)

type SchedulerManager struct{}

type scheduledTask struct {
	Name    string
	Command string
	Time    string
	Days    string
}

func taskName(name string) string {
	return `\teams-quiet-hours\` + name
}

func taskCommand(executable string, command string, browser string) string {
	return fmt.Sprintf(`"%s" %s --browser %s`, executable, command, browser)
}

func (SchedulerManager) Install(cfg config.Config, executable string, dryRun bool) error {
	tasks := []scheduledTask{
		{Name: "AllowWeekdays", Command: "allow", Time: cfg.Start, Days: "MON,TUE,WED,THU,FRI"},
		{Name: "BlockWeekdays", Command: "block", Time: cfg.End, Days: "MON,TUE,WED,THU,FRI"},
		{Name: "BlockWeekends", Command: "block", Time: "00:00", Days: "SAT,SUN"},
	}
	for _, task := range tasks {
		name := taskName(task.Name)
		args := []string{"/Create", "/TN", name, "/SC", "WEEKLY", "/D", task.Days, "/ST", task.Time, "/TR", taskCommand(executable, task.Command, cfg.Browser), "/RL", "HIGHEST", "/F"}
		if dryRun {
			fmt.Printf("DRY-RUN: schtasks %s\n", strings.Join(args, " "))
			continue
		}
		if out, err := exec.Command("schtasks", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("%s", strings.TrimSpace(string(out)))
		}
	}
	if cfg.Mode == "nuclear" {
		name := taskName("Enforce")
		args := []string{"/Create", "/TN", name, "/SC", "MINUTE", "/MO", fmt.Sprintf("%d", cfg.EnforceEveryMins), "/TR", taskCommand(executable, "enforce", cfg.Browser), "/RL", "HIGHEST", "/F"}
		if dryRun {
			fmt.Printf("DRY-RUN: schtasks %s\n", strings.Join(args, " "))
		} else if out, err := exec.Command("schtasks", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("%s", strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func (SchedulerManager) Uninstall(dryRun bool) error {
	for _, name := range []string{"AllowWeekdays", "BlockWeekdays", "BlockWeekends", "Enforce"} {
		fullName := taskName(name)
		if dryRun {
			fmt.Printf("DRY-RUN: schtasks /Delete /TN %s /F\n", fullName)
			continue
		}
		_, _ = exec.Command("schtasks", "/Delete", "/TN", fullName, "/F").CombinedOutput()
	}
	return nil
}

func (SchedulerManager) Status() ([]scheduler.Status, error) {
	var statuses []scheduler.Status
	for _, name := range []string{"AllowWeekdays", "BlockWeekdays", "BlockWeekends", "Enforce"} {
		fullName := taskName(name)
		out, err := exec.Command("schtasks", "/Query", "/TN", fullName).CombinedOutput()
		installed := err == nil
		detail := "task present"
		if err != nil {
			detail = strings.TrimSpace(string(out))
			if detail == "" {
				detail = "task absent"
			}
		}
		statuses = append(statuses, scheduler.Status{Name: fullName, Target: fullName, Installed: installed, Detail: detail})
	}
	return statuses, nil
}
