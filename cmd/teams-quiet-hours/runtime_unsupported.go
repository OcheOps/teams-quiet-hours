//go:build !linux && !darwin && !windows

package main

import (
	"fmt"

	"github.com/OcheOps/teams-quiet-hours/internal/config"
	"github.com/OcheOps/teams-quiet-hours/internal/platform"
	"github.com/OcheOps/teams-quiet-hours/internal/policy"
	"github.com/OcheOps/teams-quiet-hours/internal/scheduler"
	"github.com/OcheOps/teams-quiet-hours/internal/teams/status"
)

type unsupportedPolicy struct{}
type unsupportedScheduler struct{}
type unsupportedProcess struct{}
type unsupportedNetwork struct{}
type unsupportedTabs struct{}

func (unsupportedPolicy) Block(string, bool) error { return fmt.Errorf("unsupported platform") }
func (unsupportedPolicy) Allow(string, bool) error { return fmt.Errorf("unsupported platform") }
func (unsupportedPolicy) Status(string) ([]policy.Status, error) {
	return nil, fmt.Errorf("unsupported platform")
}

func (unsupportedScheduler) Install(config.Config, string, bool) error {
	return fmt.Errorf("unsupported platform")
}
func (unsupportedScheduler) Uninstall(bool) error { return fmt.Errorf("unsupported platform") }
func (unsupportedScheduler) Status() ([]scheduler.Status, error) {
	return nil, fmt.Errorf("unsupported platform")
}

func (unsupportedProcess) Stop(bool) error  { return fmt.Errorf("unsupported platform") }
func (unsupportedProcess) Start(bool) error { return fmt.Errorf("unsupported platform") }
func (unsupportedProcess) Status() ([]status.Item, error) {
	return nil, fmt.Errorf("unsupported platform")
}

func (unsupportedNetwork) Block(bool) error { return fmt.Errorf("unsupported platform") }
func (unsupportedNetwork) Allow(bool) error { return fmt.Errorf("unsupported platform") }
func (unsupportedNetwork) Status() ([]status.Item, error) {
	return nil, fmt.Errorf("unsupported platform")
}

func (unsupportedTabs) CloseTeamsTabs(bool) error { return fmt.Errorf("unsupported platform") }

func currentRuntime() platform.Runtime {
	return platform.Runtime{
		Name:      platform.Name(),
		Policy:    unsupportedPolicy{},
		Scheduler: unsupportedScheduler{},
		Process:   unsupportedProcess{},
		Network:   unsupportedNetwork{},
		Tabs:      unsupportedTabs{},
	}
}
