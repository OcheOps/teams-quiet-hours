//go:build darwin

package darwin

import "github.com/OcheOps/teams-quiet-hours/internal/platform"

func New() platform.Runtime {
	return platform.Runtime{
		Name:      "macOS",
		Policy:    PolicyManager{},
		Scheduler: SchedulerManager{},
		Process:   ProcessManager{},
		Network:   NetworkManager{},
		Tabs:      TabsManager{},
	}
}
