package platform

import (
	"runtime"

	"github.com/OcheOps/teams-quiet-hours/internal/policy"
	"github.com/OcheOps/teams-quiet-hours/internal/scheduler"
	"github.com/OcheOps/teams-quiet-hours/internal/teams/network"
	"github.com/OcheOps/teams-quiet-hours/internal/teams/process"
	"github.com/OcheOps/teams-quiet-hours/internal/teams/tabs"
)

type Runtime struct {
	Name      string
	Policy    policy.Manager
	Scheduler scheduler.Manager
	Process   process.Manager
	Network   network.Manager
	Tabs      tabs.Manager
}

func Name() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}
