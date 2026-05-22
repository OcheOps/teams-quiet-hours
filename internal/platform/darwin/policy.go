package darwin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/OcheOps/teams-quiet-hours/internal/policy"
)

type PolicyManager struct{}

type macBrowserTarget struct {
	Name     string
	Domain   string
	Plist    string
	Feasible bool
}

func macPrefix() string {
	return os.Getenv("TEAMS_QUIET_HOURS_SYSTEM_PREFIX")
}

func managedPreferencesDir() string {
	return filepath.Join(macPrefix(), "Library", "Managed Preferences")
}

func macPolicyTargets(browser string) ([]macBrowserTarget, error) {
	base := managedPreferencesDir()
	targets := map[string][]macBrowserTarget{
		"chrome": {
			{Name: "chrome", Domain: "com.google.Chrome", Plist: filepath.Join(base, "com.google.Chrome.plist"), Feasible: true},
		},
		"chromium": {
			{Name: "chromium", Domain: "org.chromium.Chromium", Plist: filepath.Join(base, "org.chromium.Chromium.plist"), Feasible: true},
		},
		"edge": {
			{Name: "edge", Domain: "com.microsoft.Edge", Plist: filepath.Join(base, "com.microsoft.Edge.plist"), Feasible: true},
		},
	}
	switch browser {
	case "chrome", "chromium", "edge":
		return targets[browser], nil
	case "all":
		var all []macBrowserTarget
		for _, key := range []string{"chrome", "chromium", "edge"} {
			all = append(all, targets[key]...)
		}
		return all, nil
	default:
		return nil, fmt.Errorf("unsupported browser %q", browser)
	}
}

func emptyPlist() []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict/>
</plist>
`)
}

func ensurePlist(path string, dryRun bool) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if dryRun {
		fmt.Printf("DRY-RUN: create plist %s\n", path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, emptyPlist(), 0o644)
}

func readPolicyArray(path string) ([]string, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	out, err := exec.Command("/usr/bin/plutil", "-convert", "json", "-o", "-", path).Output()
	if err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, err
	}
	raw, ok := parsed[policy.PolicyName].([]any)
	if !ok {
		return nil, nil
	}
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		if value, ok := item.(string); ok {
			values = append(values, value)
		}
	}
	return values, nil
}

func writePolicyArray(path string, values []string, dryRun bool) error {
	if dryRun {
		fmt.Printf("DRY-RUN: set %s in %s to %v\n", policy.PolicyName, path, values)
		return nil
	}
	if err := ensurePlist(path, false); err != nil {
		return err
	}
	_ = exec.Command("/usr/libexec/PlistBuddy", "-c", "Delete :"+policy.PolicyName, path).Run()
	if len(values) == 0 {
		return nil
	}
	if err := exec.Command("/usr/libexec/PlistBuddy", "-c", "Add :"+policy.PolicyName+" array", path).Run(); err != nil {
		return err
	}
	for i, value := range values {
		cmd := fmt.Sprintf("Add :%s:%d string %s", policy.PolicyName, i, value)
		if err := exec.Command("/usr/libexec/PlistBuddy", "-c", cmd, path).Run(); err != nil {
			return err
		}
	}
	return nil
}

func (PolicyManager) Block(browser string, dryRun bool) error {
	targets, err := macPolicyTargets(browser)
	if err != nil {
		return err
	}
	for _, target := range targets {
		current, err := readPolicyArray(target.Plist)
		if err != nil {
			return err
		}
		next := policy.AddTeamsURLs(current)
		if err := ensurePlist(target.Plist, dryRun); err != nil {
			return err
		}
		if err := writePolicyArray(target.Plist, next, dryRun); err != nil {
			return err
		}
		fmt.Printf("Blocked Teams notifications for %s via managed preference %s\n", target.Name, target.Plist)
	}
	return nil
}

func (PolicyManager) Allow(browser string, dryRun bool) error {
	targets, err := macPolicyTargets(browser)
	if err != nil {
		return err
	}
	for _, target := range targets {
		current, err := readPolicyArray(target.Plist)
		if err != nil {
			return err
		}
		next := policy.RemoveTeamsURLs(current)
		if len(current) == 0 {
			fmt.Printf("Allowed Teams notifications for %s; no managed preference was present\n", target.Name)
			continue
		}
		if err := writePolicyArray(target.Plist, next, dryRun); err != nil {
			return err
		}
		fmt.Printf("Allowed Teams notifications for %s by removing only Teams URL patterns\n", target.Name)
	}
	return nil
}

func (PolicyManager) Status(browser string) ([]policy.Status, error) {
	targets, err := macPolicyTargets(browser)
	if err != nil {
		return nil, err
	}
	statuses := make([]policy.Status, 0, len(targets))
	for _, target := range targets {
		values, err := readPolicyArray(target.Plist)
		blocked := err == nil && policy.ContainsAllTeamsURLs(values)
		detail := "Teams URLs absent"
		if blocked {
			detail = "Teams URLs present"
		}
		if err != nil {
			detail = err.Error()
		}
		statuses = append(statuses, policy.Status{
			Browser: target.Name,
			Target:  target.Plist,
			Blocked: blocked,
			Detail:  detail,
		})
	}
	return statuses, nil
}
