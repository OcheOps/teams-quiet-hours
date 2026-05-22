//go:build windows

package windows

import "github.com/OcheOps/teams-quiet-hours/internal/platform"

func New() platform.Runtime {
	return platform.Runtime{
		Name:      "Windows",
		Policy:    PolicyManager{},
		Scheduler: SchedulerManager{},
		Process:   ProcessManager{},
		Network:   NetworkManager{},
		Tabs:      TabsManager{},
	}
}
