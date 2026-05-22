package darwin

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OcheOps/teams-quiet-hours/internal/teams/status"
)

type ProcessManager struct{}
type NetworkManager struct{}
type TabsManager struct{}

var darwinTeamsApps = []string{"Microsoft Teams", "Microsoft Teams classic"}
var darwinTeamsProcesses = []string{"Microsoft Teams", "Teams", "MSTeams"}

func (ProcessManager) Stop(dryRun bool) error {
	for _, app := range darwinTeamsApps {
		if dryRun {
			fmt.Printf("DRY-RUN: osascript quit app %q\n", app)
			continue
		}
		_ = exec.Command("osascript", "-e", fmt.Sprintf(`tell application "%s" to quit`, app)).Run()
	}
	for _, process := range darwinTeamsProcesses {
		if dryRun {
			fmt.Printf("DRY-RUN: pkill -x %q\n", process)
			continue
		}
		_ = exec.Command("pkill", "-x", process).Run()
	}
	fmt.Println("Stopped Teams desktop apps where present.")
	return nil
}

func (ProcessManager) Start(dryRun bool) error {
	if dryRun {
		fmt.Println("DRY-RUN: open -a Microsoft Teams or open https://teams.microsoft.com/")
		return nil
	}
	if err := exec.Command("open", "-a", "Microsoft Teams").Run(); err == nil {
		return nil
	}
	return exec.Command("open", "https://teams.microsoft.com/").Start()
}

func (ProcessManager) Status() ([]status.Item, error) {
	var items []status.Item
	for _, process := range darwinTeamsProcesses {
		err := exec.Command("pgrep", "-x", process).Run()
		running := err == nil
		detail := "not running"
		if running {
			detail = "running"
		}
		items = append(items, status.Item{Name: "process", Target: process, Enabled: running, Detail: detail})
	}
	return items, nil
}

func macHostsPath() string {
	if override := os.Getenv("TEAMS_QUIET_HOURS_HOSTS_PATH"); override != "" {
		return override
	}
	return filepath.Join(macPrefix(), "etc", "hosts")
}

func macHostsBlock() string {
	return `# BEGIN teams-quiet-hours
0.0.0.0 teams.microsoft.com
0.0.0.0 statics.teams.cdn.office.net
0.0.0.0 teams.live.com
# END teams-quiet-hours
`
}

func removeMacHostsBlock(data []byte) []byte {
	start := []byte("# BEGIN teams-quiet-hours\n")
	end := []byte("# END teams-quiet-hours\n")
	s := bytes.Index(data, start)
	if s == -1 {
		return data
	}
	e := bytes.Index(data[s:], end)
	if e == -1 {
		return data
	}
	e += s + len(end)
	return append(data[:s], data[e:]...)
}

func (NetworkManager) Block(dryRun bool) error {
	path := macHostsPath()
	if dryRun {
		fmt.Printf("DRY-RUN: add teams-quiet-hours hosts block to %s\n%s", path, macHostsBlock())
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	next := append(removeMacHostsBlock(data), []byte(macHostsBlock())...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, next, 0o644)
}

func (NetworkManager) Allow(dryRun bool) error {
	path := macHostsPath()
	if dryRun {
		fmt.Printf("DRY-RUN: remove teams-quiet-hours hosts block from %s\n", path)
		return nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, removeMacHostsBlock(data), 0o644)
}

func (NetworkManager) Status() ([]status.Item, error) {
	data, err := os.ReadFile(macHostsPath())
	blocked := err == nil && strings.Contains(string(data), "# BEGIN teams-quiet-hours")
	detail := "hosts block absent"
	if blocked {
		detail = "hosts block present"
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		detail = err.Error()
	}
	return []status.Item{{Name: "network", Target: macHostsPath(), Enabled: blocked, Detail: detail}}, nil
}

func (TabsManager) CloseTeamsTabs(dryRun bool) error {
	fmt.Println("Browser tab closing is intentionally limited; Chrome/Edge remote debugging must be configured explicitly in a future release.")
	if dryRun {
		fmt.Println("DRY-RUN: would close Teams tabs via a configured remote debugging endpoint if supported")
	}
	return nil
}
