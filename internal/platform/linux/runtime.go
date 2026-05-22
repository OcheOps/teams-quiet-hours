//go:build linux

package linux

import "github.com/OcheOps/teams-quiet-hours/internal/platform"

func New() platform.Runtime {
	return platform.Runtime{
		Name:      "Linux",
		Policy:    PolicyManager{},
		Scheduler: SchedulerManager{},
		Process:   ProcessManager{},
		Network:   NetworkManager{},
		Tabs:      TabsManager{},
	}
}
