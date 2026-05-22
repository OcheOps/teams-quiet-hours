package windows

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/policy"
)

type PolicyManager struct{}

type winBrowserTarget struct {
	Name string
	Key  string
}

type registryValue struct {
	Name string
	Data string
}

type registryState map[string][]string

var regLinePattern = regexp.MustCompile(`^\s+(\S+)\s+REG_SZ\s+(.+?)\s*$`)

func windowsPolicyTargets(browser string) ([]winBrowserTarget, error) {
	targets := map[string][]winBrowserTarget{
		"chrome": {
			{Name: "chrome", Key: `HKLM\Software\Policies\Google\Chrome\` + policy.PolicyName},
		},
		"edge": {
			{Name: "edge", Key: `HKLM\Software\Policies\Microsoft\Edge\` + policy.PolicyName},
		},
		"chromium": {
			{Name: "chromium", Key: `HKLM\Software\Policies\Chromium\` + policy.PolicyName},
		},
	}
	switch browser {
	case "chrome", "edge", "chromium":
		return targets[browser], nil
	case "all":
		var all []winBrowserTarget
		for _, key := range []string{"chrome", "edge", "chromium"} {
			all = append(all, targets[key]...)
		}
		return all, nil
	default:
		return nil, fmt.Errorf("unsupported browser %q", browser)
	}
}

func statePath() string {
	return filepath.Join(config.Dir(), policy.StateFile)
}

func loadState() (registryState, error) {
	state := registryState{}
	data, err := os.ReadFile(statePath())
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	return state, json.Unmarshal(data, &state)
}

func saveState(state registryState, dryRun bool) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if dryRun {
		fmt.Printf("DRY-RUN: write %s\n%s", statePath(), string(data))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(statePath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0o644)
}

func queryRegistry(key string) ([]registryValue, error) {
	out, err := exec.Command("reg", "query", key).CombinedOutput()
	if err != nil {
		text := string(out)
		if strings.Contains(text, "unable to find") || strings.Contains(text, "cannot find") {
			return nil, nil
		}
		return nil, fmt.Errorf("%s", strings.TrimSpace(text))
	}
	var values []registryValue
	for _, line := range strings.Split(string(out), "\n") {
		matches := regLinePattern.FindStringSubmatch(line)
		if len(matches) == 3 {
			values = append(values, registryValue{Name: matches[1], Data: matches[2]})
		}
	}
	return values, nil
}

func nextRegistryName(values []registryValue, reserved map[string]bool) string {
	used := map[int]bool{}
	for _, value := range values {
		if n, err := strconv.Atoi(value.Name); err == nil {
			used[n] = true
		}
	}
	for i := 1; ; i++ {
		name := strconv.Itoa(i)
		if !used[i] && !reserved[name] {
			return name
		}
	}
}

func hasRegistryData(values []registryValue, data string) bool {
	for _, value := range values {
		if value.Data == data {
			return true
		}
	}
	return false
}

func (PolicyManager) Block(browser string, dryRun bool) error {
	targets, err := windowsPolicyTargets(browser)
	if err != nil {
		return err
	}
	state, err := loadState()
	if err != nil {
		return err
	}
	for _, target := range targets {
		values, err := queryRegistry(target.Key)
		if err != nil {
			return err
		}
		reserved := map[string]bool{}
		created := state[target.Key]
		for _, name := range created {
			reserved[name] = true
		}
		for _, url := range policy.TeamsURLs {
			if hasRegistryData(values, url) {
				continue
			}
			name := nextRegistryName(values, reserved)
			reserved[name] = true
			values = append(values, registryValue{Name: name, Data: url})
			created = append(created, name)
			if dryRun {
				fmt.Printf("DRY-RUN: reg add %s /v %s /t REG_SZ /d %s /f\n", target.Key, name, url)
				continue
			}
			if out, err := exec.Command("reg", "add", target.Key, "/v", name, "/t", "REG_SZ", "/d", url, "/f").CombinedOutput(); err != nil {
				return fmt.Errorf("%s", strings.TrimSpace(string(out)))
			}
		}
		sort.Strings(created)
		state[target.Key] = created
		fmt.Printf("Blocked Teams notifications for %s via %s\n", target.Name, target.Key)
	}
	return saveState(state, dryRun)
}

func (PolicyManager) Allow(browser string, dryRun bool) error {
	targets, err := windowsPolicyTargets(browser)
	if err != nil {
		return err
	}
	state, err := loadState()
	if err != nil {
		return err
	}
	for _, target := range targets {
		for _, name := range state[target.Key] {
			if dryRun {
				fmt.Printf("DRY-RUN: reg delete %s /v %s /f\n", target.Key, name)
				continue
			}
			_, _ = exec.Command("reg", "delete", target.Key, "/v", name, "/f").CombinedOutput()
		}
		delete(state, target.Key)
		fmt.Printf("Allowed Teams notifications for %s by deleting only registry values created by teams-quiet-hours\n", target.Name)
	}
	return saveState(state, dryRun)
}

func (PolicyManager) Status(browser string) ([]policy.Status, error) {
	targets, err := windowsPolicyTargets(browser)
	if err != nil {
		return nil, err
	}
	statuses := make([]policy.Status, 0, len(targets))
	for _, target := range targets {
		values, err := queryRegistry(target.Key)
		detail := "Teams URLs absent"
		var urls []string
		for _, value := range values {
			urls = append(urls, value.Data)
		}
		blocked := err == nil && policy.ContainsAllTeamsURLs(urls)
		if blocked {
			detail = "Teams URLs present"
		}
		if err != nil {
			detail = err.Error()
		}
		statuses = append(statuses, policy.Status{Browser: target.Name, Target: target.Key, Blocked: blocked, Detail: detail})
	}
	return statuses, nil
}
