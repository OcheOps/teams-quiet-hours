package linux

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OcheOps/teams-quiet-hours/internal/policy"
)

const policyFileName = "teams-quiet-hours.json"

type PolicyManager struct{}

type browserTarget struct {
	Name string
	Dir  string
}

func linuxPrefix() string {
	return os.Getenv("TEAMS_QUIET_HOURS_SYSTEM_PREFIX")
}

func linuxSystemPath(parts ...string) string {
	prefix := linuxPrefix()
	all := append([]string{prefix}, parts...)
	return filepath.Join(all...)
}

func linuxPolicyTargets(browser string) ([]browserTarget, error) {
	targets := map[string][]browserTarget{
		"chrome": {
			{Name: "chrome", Dir: linuxSystemPath("etc", "opt", "chrome", "policies", "managed")},
		},
		"chromium": {
			{Name: "chromium", Dir: linuxSystemPath("etc", "chromium", "policies", "managed")},
			{Name: "chromium-browser", Dir: linuxSystemPath("etc", "chromium-browser", "policies", "managed")},
		},
		"edge": {
			{Name: "edge", Dir: linuxSystemPath("etc", "opt", "edge", "policies", "managed")},
		},
	}
	switch browser {
	case "chrome", "chromium", "edge":
		return targets[browser], nil
	case "all":
		var all []browserTarget
		for _, key := range []string{"chrome", "chromium", "edge"} {
			all = append(all, targets[key]...)
		}
		return all, nil
	default:
		return nil, fmt.Errorf("unsupported browser %q", browser)
	}
}

func (PolicyManager) Block(browser string, dryRun bool) error {
	data, err := json.MarshalIndent(map[string][]string{policy.PolicyName: policy.TeamsURLs}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if !json.Valid(data) {
		return fmt.Errorf("generated policy JSON is invalid")
	}

	targets, err := linuxPolicyTargets(browser)
	if err != nil {
		return err
	}
	for _, target := range targets {
		path := filepath.Join(target.Dir, policyFileName)
		if dryRun {
			fmt.Printf("DRY-RUN: write %s\n%s", path, string(data))
			continue
		}
		if err := os.MkdirAll(target.Dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("Blocked Teams notifications for %s via %s\n", target.Name, path)
	}
	return nil
}

func (PolicyManager) Allow(browser string, dryRun bool) error {
	targets, err := linuxPolicyTargets(browser)
	if err != nil {
		return err
	}
	for _, target := range targets {
		path := filepath.Join(target.Dir, policyFileName)
		if dryRun {
			fmt.Printf("DRY-RUN: remove %s\n", path)
			continue
		}
		err := os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		fmt.Printf("Allowed Teams notifications for %s by removing %s\n", target.Name, path)
	}
	return nil
}

func (PolicyManager) Status(browser string) ([]policy.Status, error) {
	targets, err := linuxPolicyTargets(browser)
	if err != nil {
		return nil, err
	}
	statuses := make([]policy.Status, 0, len(targets))
	for _, target := range targets {
		path := filepath.Join(target.Dir, policyFileName)
		data, err := os.ReadFile(path)
		blocked := false
		detail := "policy file absent"
		if err == nil {
			var parsed map[string][]string
			if json.Unmarshal(data, &parsed) == nil && policy.ContainsAllTeamsURLs(parsed[policy.PolicyName]) {
				blocked = true
				detail = "Teams URLs present"
			} else {
				detail = "policy file exists but does not match expected Teams policy"
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			detail = err.Error()
		}
		statuses = append(statuses, policy.Status{
			Browser: target.Name,
			Target:  path,
			Blocked: blocked,
			Detail:  detail,
		})
	}
	return statuses, nil
}
