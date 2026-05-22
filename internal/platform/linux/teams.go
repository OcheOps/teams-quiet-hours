package linux

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

var linuxTeamsProcesses = []string{"teams", "ms-teams", "msteams"}

func (ProcessManager) Stop(dryRun bool) error {
	for _, name := range linuxTeamsProcesses {
		if dryRun {
			fmt.Printf("DRY-RUN: pkill -x %s\n", name)
			continue
		}
		_ = exec.Command("pkill", "-x", name).Run()
	}
	fmt.Println("Stopped Teams desktop processes where present.")
	return nil
}

func (ProcessManager) Start(dryRun bool) error {
	if dryRun {
		fmt.Println("DRY-RUN: xdg-open https://teams.microsoft.com/")
		return nil
	}
	if err := exec.Command("xdg-open", "https://teams.microsoft.com/").Start(); err != nil {
		return fmt.Errorf("could not launch Teams web with xdg-open: %w", err)
	}
	return nil
}

func (ProcessManager) Status() ([]status.Item, error) {
	var items []status.Item
	for _, name := range linuxTeamsProcesses {
		err := exec.Command("pgrep", "-x", name).Run()
		running := err == nil
		detail := "not running"
		if running {
			detail = "running"
		}
		items = append(items, status.Item{Name: "process", Target: name, Enabled: running, Detail: detail})
	}
	return items, nil
}

func hostsPath() string {
	if override := os.Getenv("TEAMS_QUIET_HOURS_HOSTS_PATH"); override != "" {
		return override
	}
	return filepath.Join(linuxPrefix(), "etc", "hosts")
}

func teamsHostsBlock() string {
	return `# BEGIN teams-quiet-hours
0.0.0.0 teams.microsoft.com
0.0.0.0 statics.teams.cdn.office.net
0.0.0.0 teams.live.com
# END teams-quiet-hours
`
}

func removeHostsBlock(data []byte) []byte {
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
	path := hostsPath()
	if dryRun {
		fmt.Printf("DRY-RUN: add teams-quiet-hours hosts block to %s\n%s", path, teamsHostsBlock())
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	next := removeHostsBlock(data)
	next = append(next, []byte(teamsHostsBlock())...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, next, 0o644)
}

func (NetworkManager) Allow(dryRun bool) error {
	path := hostsPath()
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
	return os.WriteFile(path, removeHostsBlock(data), 0o644)
}

func (NetworkManager) Status() ([]status.Item, error) {
	data, err := os.ReadFile(hostsPath())
	blocked := err == nil && strings.Contains(string(data), "# BEGIN teams-quiet-hours")
	detail := "hosts block absent"
	if blocked {
		detail = "hosts block present"
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		detail = err.Error()
	}
	return []status.Item{{Name: "network", Target: hostsPath(), Enabled: blocked, Detail: detail}}, nil
}

func (TabsManager) CloseTeamsTabs(dryRun bool) error {
	fmt.Println("Browser tab closing requires Chrome/Edge remote debugging and is not enabled by default.")
	if dryRun {
		fmt.Println("DRY-RUN: would close Teams tabs via a configured remote debugging endpoint if supported")
	}
	return nil
}
