package windows

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/OcheOps/teams-quiet-hours/internal/teams/status"
)

type ProcessManager struct{}
type NetworkManager struct{}
type TabsManager struct{}

const windowsFirewallRuleName = "teams-quiet-hours Teams block"

var windowsTeamsProcesses = []string{"ms-teams", "Teams", "msteams"}

func powershell(script string) ([]byte, error) {
	return exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script).CombinedOutput()
}

func (ProcessManager) Stop(dryRun bool) error {
	for _, name := range windowsTeamsProcesses {
		script := fmt.Sprintf(`Get-Process -Name '%s' -ErrorAction SilentlyContinue | Stop-Process -Force`, name)
		if dryRun {
			fmt.Printf("DRY-RUN: powershell %s\n", script)
			continue
		}
		_, _ = powershell(script)
	}
	fmt.Println("Stopped Teams desktop processes where present.")
	return nil
}

func (ProcessManager) Start(dryRun bool) error {
	script := `Start-Process "https://teams.microsoft.com/"`
	if dryRun {
		fmt.Printf("DRY-RUN: powershell %s\n", script)
		return nil
	}
	out, err := powershell(script)
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (ProcessManager) Status() ([]status.Item, error) {
	var items []status.Item
	for _, name := range windowsTeamsProcesses {
		script := fmt.Sprintf(`if (Get-Process -Name '%s' -ErrorAction SilentlyContinue) { 'running' } else { 'not running' }`, name)
		out, err := powershell(script)
		detail := strings.TrimSpace(string(out))
		running := err == nil && detail == "running"
		if detail == "" {
			detail = "not running"
		}
		items = append(items, status.Item{Name: "process", Target: name, Enabled: running, Detail: detail})
	}
	return items, nil
}

func (NetworkManager) Block(dryRun bool) error {
	script := fmt.Sprintf(`if (-not (Get-NetFirewallRule -DisplayName '%s' -ErrorAction SilentlyContinue)) { New-NetFirewallRule -DisplayName '%s' -Direction Outbound -Action Block -RemoteFqdn 'teams.microsoft.com' | Out-Null }`, windowsFirewallRuleName, windowsFirewallRuleName)
	if dryRun {
		fmt.Printf("DRY-RUN: powershell %s\n", script)
		return nil
	}
	out, err := powershell(script)
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (NetworkManager) Allow(dryRun bool) error {
	script := fmt.Sprintf(`Get-NetFirewallRule -DisplayName '%s' -ErrorAction SilentlyContinue | Remove-NetFirewallRule`, windowsFirewallRuleName)
	if dryRun {
		fmt.Printf("DRY-RUN: powershell %s\n", script)
		return nil
	}
	_, _ = powershell(script)
	return nil
}

func (NetworkManager) Status() ([]status.Item, error) {
	script := fmt.Sprintf(`if (Get-NetFirewallRule -DisplayName '%s' -ErrorAction SilentlyContinue) { 'present' } else { 'absent' }`, windowsFirewallRuleName)
	out, err := powershell(script)
	detail := strings.TrimSpace(string(out))
	blocked := err == nil && detail == "present"
	if detail == "" {
		detail = "absent"
	}
	return []status.Item{{Name: "network", Target: windowsFirewallRuleName, Enabled: blocked, Detail: detail}}, nil
}

func (TabsManager) CloseTeamsTabs(dryRun bool) error {
	fmt.Println("Browser tab closing requires a configured browser debugging endpoint and is not enabled by default.")
	if dryRun {
		fmt.Println("DRY-RUN: would close Teams tabs via a configured remote debugging endpoint if supported")
	}
	return nil
}
